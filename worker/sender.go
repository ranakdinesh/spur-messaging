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
	providerRegistry *services.ProviderRegistry
}

func NewSender(queue ports.MessageQueue, messageRepo ports.MessageRepository, providerRegistry *services.ProviderRegistry) *Sender {
	return &Sender{
		queue:            queue,
		messageRepo:      messageRepo,
		providerRegistry: providerRegistry,
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
		lastErr = s.send(sendCtx, msg)
		cancel()

		if lastErr == nil {
			return nil
		}

		// Section 10A.3: On provider 429
		// Placeholder for 429 handling with Retry-After

		// Section 10A.3: On provider 401/403
		// Placeholder for credential error handling
	}

	// If we are here, all retries failed (Section 10A.3)
	failReason := lastErr.Error()
	msg.Status = domain.MessageStatusFailed
	msg.ErrorMessage = &failReason
	now := time.Now()
	msg.FailedAt = &now

	return s.messageRepo.UpdateStatus(ctx, msg.TenantID, msg.ID, msg.Status, "")
}

func (s *Sender) send(ctx context.Context, msg *domain.Message) error {
	provider, cfg, err := s.providerRegistry.GetProvider(ctx, msg.TenantID, msg.Channel)
	if err != nil {
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
		// In a real implementation, we'd render the template before enqueuing or here.
		// AGENTS.md Section 12.10 says services render template before enqueuing.

		result, err = emailProvider.SendEmail(ctx, cfg, req)

	case domain.ChannelWhatsApp, domain.ChannelSMS:
		req := ports.ProviderSendRequest{
			Recipient:    msg.Recipient,
			MessageType:  msg.MessageType,
			TemplateName: msg.TemplateName,
			// ... other fields
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

	return s.messageRepo.UpdateStatus(ctx, msg.TenantID, msg.ID, msg.Status, msg.ProviderMessageID)
}
