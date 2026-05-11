package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
	"github.com/ranakdinesh/spur-messaging/core/services"
)

type Sender struct {
	queue            ports.MessageQueue
	messageRepo      ports.MessageRepository
	campaignRepo     ports.CampaignRepository
	providerRepo     ports.ProviderConfigRepository
	providerRegistry *services.ProviderRegistry
	billingSvc       ports.BillingService
}

func NewSender(
	queue ports.MessageQueue,
	messageRepo ports.MessageRepository,
	campaignRepo ports.CampaignRepository,
	providerRepo ports.ProviderConfigRepository,
	providerRegistry *services.ProviderRegistry,
	billingSvc ports.BillingService,
) *Sender {
	return &Sender{
		queue:            queue,
		messageRepo:      messageRepo,
		campaignRepo:     campaignRepo,
		providerRepo:     providerRepo,
		providerRegistry: providerRegistry,
		billingSvc:       billingSvc,
	}
}

func (s *Sender) Start(ctx context.Context) error {
	return s.queue.StartConsumer(ctx, s.HandleMessage)
}

func (s *Sender) HandleMessage(ctx context.Context, qmsg ports.QueueMessage) error {
	// Load message from DB
	msg, err := s.messageRepo.GetByID(ctx, qmsg.TenantID, qmsg.MessageID)
	if err != nil {
		return err
	}

	// Resolve provider config once
	_, cfg, err := s.providerRegistry.GetProvider(ctx, msg.TenantID, msg.Channel)
	if err != nil {
		return err
	}

	// Retry logic with exponential backoff (Section 10A.3)
	// 5s -> 30s -> 120s (max 3 retries)
	backoffs := []time.Duration{5 * time.Second, 30 * time.Second, 120 * time.Second}
	var lastErr error

	for attempt := 0; attempt <= len(backoffs); attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoffs[attempt-1]):
			}
		}

		// Set 30-second HTTP timeout (Section 10A.3)
		sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		lastErr = s.sendWithCfg(sendCtx, msg, cfg)
		cancel()

		if lastErr == nil {
			return nil
		}

		// Section 10A.3: On provider 429
		if errors.Is(lastErr, domain.ErrRateLimitExceeded) {
			// In a real scenario, we might extract Retry-After from a custom error type
			// For now, let's use a default or simulated value
			retryAfter := 60 * time.Second
			// Log rate limit
			fmt.Printf("Rate limit exceeded for tenant %s, channel %s. Retrying after %v\n", msg.TenantID, msg.Channel, retryAfter)
			time.Sleep(retryAfter)
			attempt-- // Don't count 429 as a failed attempt for max retries
			continue
		}

		// Section 10A.3: On provider 401/403
		if errors.Is(lastErr, domain.ErrUnauthorized) || errors.Is(lastErr, domain.ErrForbidden) || errors.Is(lastErr, domain.ErrCredentialsExpired) {
			// Set provider_config.is_active = false
			if cfg != nil {
				_ = s.providerRepo.UpdateIsActive(ctx, msg.TenantID, cfg.ID, false)
			}
			fmt.Printf("Provider credentials expired for tenant %s\n", msg.TenantID)

			msg.Status = domain.MessageStatusFailed
			errCode := "CREDENTIALS_EXPIRED"
			msg.ErrorCode = &errCode
			now := time.Now()
			msg.FailedAt = &now
			_ = s.messageRepo.UpdateStatus(ctx, msg.TenantID, msg.ID, msg.Status, "")

			return nil // Do not retry
		}
	}

	// If we are here, all retries failed (Section 10A.3)
	failReason := lastErr.Error()
	msg.Status = domain.MessageStatusFailed
	msg.ErrorMessage = &failReason
	errCode := "PROVIDER_ERROR"
	msg.ErrorCode = &errCode
	now := time.Now()
	msg.FailedAt = &now

	err = s.messageRepo.UpdateStatus(ctx, msg.TenantID, msg.ID, msg.Status, "")

	// For campaigns: increment campaign stats.failed counter
	if msg.CampaignID != nil {
		campaign, err := s.campaignRepo.GetByID(ctx, msg.TenantID, *msg.CampaignID)
		if err == nil {
			campaign.Stats.Failed++
			_ = s.campaignRepo.UpdateStats(ctx, msg.TenantID, campaign.ID, campaign.Stats)
		}
	}

	return err
}

func (s *Sender) sendWithCfg(ctx context.Context, msg *domain.Message, cfg *domain.ProviderConfig) error {
	provider, _, err := s.providerRegistry.GetProvider(ctx, msg.TenantID, msg.Channel)
	if err != nil {
		return err
	}

	if err := s.messageRepo.UpdateStatus(ctx, msg.TenantID, msg.ID, domain.MessageStatusProviderSubmitted, ""); err != nil {
		return err
	}

	var result *ports.ProviderSendResult

	switch msg.Channel {
	case domain.ChannelEmail:
		emailProvider, ok := provider.(ports.EmailProvider)
		if !ok {
			return errors.New("provider does not support EmailProvider interface")
		}

		// Construct EmailSendRequest from msg
		var body string
		if msg.TextBody != nil {
			body = *msg.TextBody
		}
		req := ports.EmailSendRequest{
			To:          msg.Recipient,
			Subject:     msg.Metadata["subject"],
			HTMLBody:    body,
			TrackOpens:  true,
			TrackClicks: true,
			Metadata:    map[string]string{"message_id": msg.ID.String(), "tenant_id": msg.TenantID.String()},
		}

		result, err = emailProvider.SendEmail(ctx, cfg, req)

	case domain.ChannelWhatsApp, domain.ChannelSMS:
		req := ports.ProviderSendRequest{
			Recipient:    msg.Recipient,
			MessageType:  msg.MessageType,
			TemplateName: msg.TemplateName,
		}
		if msg.TextBody != nil {
			req.Text = msg.TextBody
		}
		if msg.TemplateParams != nil {
			req.TemplateParams = msg.TemplateParams
		}

		result, err = provider.Send(ctx, cfg, req)

	default:
		return fmt.Errorf("unsupported channel: %s", msg.Channel)
	}

	if err != nil {
		return err
	}

	// Update message status in DB
	msg.Status = result.Status
	msg.ProviderMessageID = result.ProviderMessageID
	msg.Cost = result.Cost
	msg.SentAt = &result.Timestamp

	if err := s.messageRepo.UpdateStatus(ctx, msg.TenantID, msg.ID, msg.Status, msg.ProviderMessageID); err != nil {
		return err
	}
	if s.billingSvc != nil && msg.Direction == "outbound" {
		_, err := s.billingSvc.RecordMessageCharge(ctx, domain.UsageCharge{
			TenantID:     msg.TenantID,
			MessageID:    msg.ID,
			CampaignID:   msg.CampaignID,
			Channel:      msg.Channel,
			Category:     billingCategoryForMessage(msg),
			Country:      msg.Metadata["country"],
			Currency:     msg.Metadata["currency"],
			ProviderCost: result.Cost,
			Provider:     cfg.Provider,
			Description:  "Message accepted by provider",
			OccurredAt:   result.Timestamp,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func billingCategoryForMessage(msg *domain.Message) string {
	if msg.Metadata != nil {
		if category := msg.Metadata["category"]; category != "" {
			return category
		}
	}
	if msg.Channel == domain.ChannelWhatsApp && msg.MessageType == domain.MessageTypeTemplate {
		return "utility"
	}
	if msg.Channel == domain.ChannelEmail {
		return "marketing"
	}
	return "service"
}
