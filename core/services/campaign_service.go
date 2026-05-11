package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type CampaignService struct {
	repo           ports.CampaignRepository
	templateRepo   ports.TemplateRepository
	segmentRepo    ports.SegmentRepository
	queue          ports.MessageQueue
	suppressionSvc ports.SuppressionService
	unsubscribeSvc ports.UnsubscribeService
	contactRepo    ports.ContactRepository
}

func NewCampaignService(
	repo ports.CampaignRepository,
	templateRepo ports.TemplateRepository,
	segmentRepo ports.SegmentRepository,
	queue ports.MessageQueue,
	suppressionSvc ports.SuppressionService,
	unsubscribeSvc ports.UnsubscribeService,
	contactRepo ports.ContactRepository,
) *CampaignService {
	return &CampaignService{
		repo:           repo,
		templateRepo:   templateRepo,
		segmentRepo:    segmentRepo,
		queue:          queue,
		suppressionSvc: suppressionSvc,
		unsubscribeSvc: unsubscribeSvc,
		contactRepo:    contactRepo,
	}
}

func (s *CampaignService) Create(ctx context.Context, tenantID uuid.UUID, req ports.CreateCampaignRequest) (*domain.Campaign, error) {
	campaign := &domain.Campaign{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Name:           req.Name,
		Channel:        req.Channel,
		TemplateID:     req.TemplateID,
		TemplateParams: req.TemplateParams,
		SegmentID:      req.SegmentID,
		ContactIDs:     req.ContactIDs,
		ScheduledAt:    req.ScheduledAt,
		Status:         domain.CampaignStatusDraft,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err := s.repo.Create(ctx, campaign)
	if err != nil {
		return nil, err
	}

	return campaign, nil
}

func (s *CampaignService) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Campaign, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *CampaignService) List(ctx context.Context, tenantID uuid.UUID, status *domain.CampaignStatus, page, perPage int) ([]domain.Campaign, int, error) {
	return s.repo.List(ctx, tenantID, status, page, perPage)
}

func (s *CampaignService) Update(ctx context.Context, tenantID, id uuid.UUID, req ports.UpdateCampaignRequest) (*domain.Campaign, error) {
	campaign, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Section 10A.2: only draft/scheduled campaigns → ErrInvalidInput
	if campaign.Status != domain.CampaignStatusDraft && campaign.Status != domain.CampaignStatusScheduled {
		return nil, domain.NewValidationError("status", "cannot update running/completed campaign")
	}

	if req.Name != nil {
		campaign.Name = *req.Name
	}
	if req.TemplateParams != nil {
		campaign.TemplateParams = *req.TemplateParams
	}
	if req.SegmentID != nil {
		campaign.SegmentID = req.SegmentID
	}
	if req.ContactIDs != nil {
		campaign.ContactIDs = *req.ContactIDs
	}
	if req.ScheduledAt != nil {
		campaign.ScheduledAt = req.ScheduledAt
	}
	campaign.UpdatedAt = time.Now()

	err = s.repo.Update(ctx, campaign)
	if err != nil {
		return nil, err
	}

	return campaign, nil
}

func (s *CampaignService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	campaign, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	// Section 10A.2: only draft campaigns → ErrInvalidInput
	if campaign.Status != domain.CampaignStatusDraft {
		return domain.NewValidationError("status", "only draft campaigns can be deleted")
	}
	return s.repo.Delete(ctx, tenantID, id)
}

func (s *CampaignService) Execute(ctx context.Context, tenantID, id uuid.UUID) error {
	campaign, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	// Section 10A.2: Campaign status check - draft or scheduled only
	if campaign.Status != domain.CampaignStatusDraft && campaign.Status != domain.CampaignStatusScheduled {
		return domain.ErrCampaignNotExecutable
	}

	// Section 10A.2: Referenced template exists AND approved
	tmpl, err := s.templateRepo.GetByID(ctx, tenantID, campaign.TemplateID)
	if err != nil {
		return domain.ErrNotFound
	}
	if campaign.Channel == domain.ChannelWhatsApp && tmpl.Status != domain.TemplateStatusApproved {
		return domain.ErrTemplateNotApproved
	}

	// Resolve contacts
	var contacts []domain.Contact
	if campaign.SegmentID != nil {
		// Section 10A.2: Referenced segment exists
		contacts, _, err = s.segmentRepo.ResolveContacts(ctx, tenantID, *campaign.SegmentID, 1, 1000000)
		if err != nil {
			return err
		}
	} else if len(campaign.ContactIDs) > 0 {
		for _, contactID := range campaign.ContactIDs {
			contact, err := s.contactRepo.GetByID(ctx, tenantID, contactID)
			if err == nil {
				contacts = append(contacts, *contact)
			}
		}
	}

	// Section 10A.2: Segment resolves to > 0 contacts
	if len(contacts) == 0 {
		return domain.NewValidationError("segment", "segment has no contacts")
	}

	// Update status to running
	campaign.Status = domain.CampaignStatusRunning
	campaign.StartedAt = new(time.Time)
	*campaign.StartedAt = time.Now()
	err = s.repo.UpdateStatus(ctx, tenantID, id, campaign.Status)
	if err != nil {
		return err
	}

	// Fan out messages
	var queueMsgs []ports.QueueMessage
	for _, contact := range contacts {
		// Section 10A.2: Filter out non-opted-in
		var optedIn bool
		switch campaign.Channel {
		case domain.ChannelWhatsApp:
			optedIn = contact.OptInWhatsApp == domain.OptInStatusOptedIn
		case domain.ChannelSMS:
			optedIn = contact.OptInSMS == domain.OptInStatusOptedIn
		case domain.ChannelEmail:
			optedIn = contact.OptInEmail == domain.OptInStatusOptedIn
		}
		if !optedIn {
			continue
		}

		recipient := ""
		if contact.Phone != nil {
			recipient = *contact.Phone
		}
		if campaign.Channel == domain.ChannelEmail && contact.Email != nil {
			recipient = *contact.Email
		}

		if recipient == "" {
			continue
		}

		if s.suppressionSvc != nil {
			suppressed, err := s.suppressionSvc.IsSuppressed(ctx, tenantID, campaign.Channel, recipient)
			if err != nil || suppressed {
				continue
			}
		}
		if campaign.Channel == domain.ChannelEmail && s.unsubscribeSvc != nil {
			unsubscribed, err := s.unsubscribeSvc.IsUnsubscribed(ctx, tenantID, recipient)
			if err != nil || unsubscribed {
				continue
			}
		}

		msg := &domain.Message{
			ID:             uuid.New(),
			TenantID:       tenantID,
			CampaignID:     &campaign.ID,
			Channel:        campaign.Channel,
			Direction:      "outbound",
			Recipient:      recipient,
			MessageType:    domain.MessageTypeTemplate,
			TemplateID:     &campaign.TemplateID,
			TemplateParams: campaign.TemplateParams,
			Status:         domain.MessageStatusQueued,
			CreatedAt:      time.Now(),
		}

		queueMsgs = append(queueMsgs, ports.QueueMessage{
			MessageID: msg.ID,
			TenantID:  tenantID,
			Channel:   campaign.Channel,
			Priority:  0,
		})
	}

	// Enqueue in bulk
	if len(queueMsgs) > 0 {
		err = s.queue.EnqueueBulk(ctx, queueMsgs)
		if err != nil {
			// Section 10A.3: Redis unavailable during fan-out
			campaign.Status = domain.CampaignStatusFailed
			_ = s.repo.UpdateStatus(ctx, tenantID, id, campaign.Status)
			return domain.ErrQueueUnavailable
		}
	}

	// Update stats
	campaign.Stats.Total = len(queueMsgs)
	campaign.Stats.Queued = len(queueMsgs)
	err = s.repo.UpdateStats(ctx, tenantID, id, campaign.Stats)
	if err != nil {
		return err
	}

	return nil
}

func (s *CampaignService) Pause(ctx context.Context, tenantID, id uuid.UUID) error {
	campaign, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	// Section 10A.2: only running campaigns → ErrInvalidInput
	if campaign.Status != domain.CampaignStatusRunning {
		return domain.NewValidationError("status", "can only pause running campaigns")
	}
	return s.repo.UpdateStatus(ctx, tenantID, id, domain.CampaignStatusPaused)
}

func (s *CampaignService) Resume(ctx context.Context, tenantID, id uuid.UUID) error {
	campaign, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	// Section 10A.2: only paused campaigns → ErrInvalidInput
	if campaign.Status != domain.CampaignStatusPaused {
		return domain.NewValidationError("status", "can only resume paused campaigns")
	}
	return s.repo.UpdateStatus(ctx, tenantID, id, domain.CampaignStatusRunning)
}

func (s *CampaignService) GetStats(ctx context.Context, tenantID, id uuid.UUID) (*domain.CampaignStats, error) {
	campaign, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	return &campaign.Stats, nil
}
