package worker

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type CampaignExecutor struct {
	campaignRepo   ports.CampaignRepository
	contactRepo    ports.ContactRepository
	segmentRepo    ports.SegmentRepository
	templateRepo   ports.TemplateRepository
	suppressionSvc ports.SuppressionService
	unsubscribeSvc ports.UnsubscribeService
	messageRepo    ports.MessageRepository
	queue          ports.MessageQueue
}

func NewCampaignExecutor(
	campaignRepo ports.CampaignRepository,
	contactRepo ports.ContactRepository,
	segmentRepo ports.SegmentRepository,
	templateRepo ports.TemplateRepository,
	suppressionSvc ports.SuppressionService,
	unsubscribeSvc ports.UnsubscribeService,
	messageRepo ports.MessageRepository,
	queue ports.MessageQueue,
) *CampaignExecutor {
	return &CampaignExecutor{
		campaignRepo:   campaignRepo,
		contactRepo:    contactRepo,
		segmentRepo:    segmentRepo,
		templateRepo:   templateRepo,
		suppressionSvc: suppressionSvc,
		unsubscribeSvc: unsubscribeSvc,
		messageRepo:    messageRepo,
		queue:          queue,
	}
}

func (e *CampaignExecutor) Start(ctx context.Context) {
	// Section 10A.3: Crash recovery on startup
	e.RecoverRunningCampaigns(ctx)

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.ProcessScheduledCampaigns(ctx)
		}
	}
}

func (e *CampaignExecutor) ProcessScheduledCampaigns(ctx context.Context) {
	campaigns, err := e.campaignRepo.GetScheduledCampaigns(ctx, time.Now())
	if err != nil {
		return
	}

	for _, campaign := range campaigns {
		go e.ExecuteCampaign(ctx, campaign)
	}
}

func (e *CampaignExecutor) ExecuteCampaign(ctx context.Context, campaign domain.Campaign) {
	// Section 10A.3: Template deleted/deactivated re-check
	tmpl, err := e.templateRepo.GetByID(ctx, campaign.TenantID, campaign.TemplateID)
	if err != nil || tmpl.Status != domain.TemplateStatusApproved {
		_ = e.campaignRepo.UpdateStatus(ctx, campaign.TenantID, campaign.ID, domain.CampaignStatusFailed)
		// We could store the error message if UpdateStatus supported it or use metadata
		return
	}

	// Update status to running
	_ = e.campaignRepo.UpdateStatus(ctx, campaign.TenantID, campaign.ID, domain.CampaignStatusRunning)

	var contacts []domain.Contact
	stats := domain.CampaignStats{}

	if campaign.SegmentID != nil {
		// For simplicity, fetch all contacts in the segment. In real scenario, paginate.
		contacts, _, err = e.segmentRepo.ResolveContacts(ctx, campaign.TenantID, *campaign.SegmentID, 1, 1000000)
	} else if len(campaign.ContactIDs) > 0 {
		for _, id := range campaign.ContactIDs {
			contact, err := e.contactRepo.GetByID(ctx, campaign.TenantID, id)
			if err != nil {
				// Section 10A.3: Contact deleted mid-campaign
				stats.Failed++ // Using Failed as "skipped"
				continue
			}
			contacts = append(contacts, *contact)
		}
	}

	if err != nil {
		_ = e.campaignRepo.UpdateStatus(ctx, campaign.TenantID, campaign.ID, domain.CampaignStatusFailed)
		return
	}

	stats.Total = len(contacts)

	for _, contact := range contacts {
		// Section 10A.3: Duplicate send prevention
		recipient := e.getRecipient(campaign.Channel, contact)
		if recipient == "" {
			stats.Failed++
			continue
		}
		exists, _ := e.messageRepo.ExistsForCampaign(ctx, campaign.ID, recipient)
		if exists {
			continue // Already sent or queued
		}

		// 1. Check suppression for every channel and unsubscribe for email.
		if e.suppressionSvc != nil {
			suppressed, _ := e.suppressionSvc.IsSuppressed(ctx, campaign.TenantID, campaign.Channel, recipient)
			if suppressed {
				stats.Failed++
				continue
			}
		}
		if campaign.Channel == domain.ChannelEmail && e.unsubscribeSvc != nil {
			unsubscribed, _ := e.unsubscribeSvc.IsUnsubscribed(ctx, campaign.TenantID, recipient)
			if unsubscribed {
				stats.Failed++
				continue
			}
		}

		// 2. Create message
		msg := &domain.Message{
			ID:             uuid.New(),
			TenantID:       campaign.TenantID,
			CampaignID:     &campaign.ID,
			Channel:        campaign.Channel,
			Direction:      "outbound",
			Recipient:      e.getRecipient(campaign.Channel, contact),
			MessageType:    domain.MessageTypeTemplate,
			TemplateID:     &campaign.TemplateID,
			TemplateParams: campaign.TemplateParams,
			Status:         domain.MessageStatusQueued,
			CreatedAt:      time.Now(),
		}

		if err := e.messageRepo.Create(ctx, msg); err != nil {
			stats.Failed++
			continue
		}

		// 3. Enqueue
		qmsg := ports.QueueMessage{
			MessageID: msg.ID,
			TenantID:  msg.TenantID,
			Channel:   msg.Channel,
			Priority:  0,
		}

		if err := e.queue.Enqueue(ctx, qmsg); err != nil {
			// Section 10A.3: Redis unavailable during fan-out
			_ = e.campaignRepo.UpdateStatus(ctx, campaign.TenantID, campaign.ID, domain.CampaignStatusFailed)
			return
		}

		stats.Queued++
	}

	// Final status update
	_ = e.campaignRepo.UpdateStatus(ctx, campaign.TenantID, campaign.ID, domain.CampaignStatusCompleted)
	_ = e.campaignRepo.UpdateStats(ctx, campaign.TenantID, campaign.ID, stats)
}

func (e *CampaignExecutor) RecoverRunningCampaigns(ctx context.Context) {
	campaigns, err := e.campaignRepo.GetRunningCampaigns(ctx)
	if err != nil {
		return
	}

	for _, campaign := range campaigns {
		go e.ExecuteCampaign(ctx, campaign)
	}
}

func (e *CampaignExecutor) getRecipient(channel domain.Channel, contact domain.Contact) string {
	switch channel {
	case domain.ChannelEmail:
		if contact.Email != nil {
			return *contact.Email
		}
	case domain.ChannelWhatsApp, domain.ChannelSMS:
		if contact.Phone != nil {
			return *contact.Phone
		}
	}
	return ""
}
