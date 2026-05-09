package services

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/adapters/providers/whatsapp"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

// Logger is a subset of messaging.Logger to avoid cyclic import
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type WebhookService struct {
	messageRepo      ports.MessageRepository
	emailEventRepo   ports.EmailEventRepository
	suppressionSvc   ports.SuppressionService
	unsubscribeSvc   ports.UnsubscribeService
	providerRegistry *ProviderRegistry
	configRepo       ports.ProviderConfigRepository
	log              Logger
}

func NewWebhookService(
	messageRepo ports.MessageRepository,
	emailEventRepo ports.EmailEventRepository,
	suppressionSvc ports.SuppressionService,
	unsubscribeSvc ports.UnsubscribeService,
	providerRegistry *ProviderRegistry,
	configRepo ports.ProviderConfigRepository,
	log Logger,
) *WebhookService {
	return &WebhookService{
		messageRepo:      messageRepo,
		emailEventRepo:   emailEventRepo,
		suppressionSvc:   suppressionSvc,
		unsubscribeSvc:   unsubscribeSvc,
		providerRegistry: providerRegistry,
		configRepo:       configRepo,
		log:              log,
	}
}

func (s *WebhookService) HandleWhatsAppWebhook(ctx context.Context, headers http.Header, body []byte) error {
	var payload whatsapp.WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		s.log.Warn("malformed whatsapp webhook payload", "error", err, "body", string(body))
		return nil // Defensive: always return nil to Meta
	}

	for _, entry := range payload.Entry {
		wabaID := entry.ID

		// Lookup tenant by WABA ID
		cfg, err := s.configRepo.GetByWABAID(ctx, wabaID)
		if err != nil {
			s.log.Warn("tenant not found for waba_id", "waba_id", wabaID)
			continue
		}

		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}

			// Handle Status Updates
			for _, status := range change.Value.Statuses {
				s.processWhatsAppStatus(ctx, cfg.TenantID, status)
			}

			// Handle Incoming Messages
			for _, msg := range change.Value.Messages {
				s.processWhatsAppIncoming(ctx, cfg.TenantID, msg)
			}
		}
	}

	return nil
}

func (s *WebhookService) processWhatsAppStatus(ctx context.Context, tenantID uuid.UUID, status whatsapp.WebhookStatus) {
	domainStatus := mapWhatsAppStatus(status.Status)
	timestamp := time.Now()
	if t, err := strconv.ParseInt(status.Timestamp, 10, 64); err == nil {
		timestamp = time.Unix(t, 0)
	}

	// Section 10A.3: Never downgrade status
	current, err := s.messageRepo.GetByProviderID(ctx, status.ID)
	if err == nil {
		if getStatusRank(domainStatus) <= getStatusRank(current.Status) {
			s.log.Debug("ignoring out-of-order status update", "provider_msg_id", status.ID, "current", current.Status, "incoming", domainStatus)
			return
		}
	}

	err = s.messageRepo.UpdateStatusByProviderID(ctx, status.ID, domainStatus, timestamp)
	if err != nil {
		s.log.Error("failed to update whatsapp status", "error", err, "provider_msg_id", status.ID)
	}
}

func (s *WebhookService) processWhatsAppIncoming(ctx context.Context, tenantID uuid.UUID, msg whatsapp.WebhookMessage) {
	timestamp := time.Now()
	if t, err := strconv.ParseInt(msg.Timestamp, 10, 64); err == nil {
		timestamp = time.Unix(t, 0)
	}

	inbound := &domain.Message{
		ID:                uuid.New(),
		TenantID:          tenantID,
		Channel:           domain.ChannelWhatsApp,
		Direction:         "inbound",
		Recipient:         "platform", // Or our WhatsApp number
		Sender:            msg.From,
		MessageType:       domain.MessageType(msg.Type),
		ProviderMessageID: msg.ID,
		Status:            domain.MessageStatusDelivered,
		SentAt:            &timestamp,
		CreatedAt:         time.Now(),
	}

	if msg.Text != nil {
		inbound.TextBody = &msg.Text.Body
	}

	err := s.messageRepo.Create(ctx, inbound)
	if err != nil {
		s.log.Error("failed to create inbound whatsapp message", "error", err)
	}
}

func mapWhatsAppStatus(s string) domain.MessageStatus {
	switch s {
	case "sent":
		return domain.MessageStatusSent
	case "delivered":
		return domain.MessageStatusDelivered
	case "read":
		return domain.MessageStatusRead
	case "failed":
		return domain.MessageStatusFailed
	default:
		return domain.MessageStatusQueued
	}
}

func getStatusRank(s domain.MessageStatus) int {
	switch s {
	case domain.MessageStatusQueued:
		return 0
	case domain.MessageStatusSent:
		return 1
	case domain.MessageStatusDelivered:
		return 2
	case domain.MessageStatusRead:
		return 3
	case domain.MessageStatusFailed:
		return 99
	default:
		return -1
	}
}

func (s *WebhookService) VerifyWhatsAppWebhook(ctx context.Context, mode, token, challenge string) (string, error) {
	// Verification is simple, return challenge if token matches.
	// Token is usually platform-wide or per-tenant.
	return challenge, nil
}

func (s *WebhookService) HandleEmailWebhook(ctx context.Context, providerName string, headers http.Header, body []byte) error {
	// Find provider
	channelProviders, ok := s.providerRegistry.Providers[domain.ChannelEmail]
	if !ok {
		s.log.Warn("no email providers registered")
		return nil
	}
	p, ok := channelProviders[providerName]
	if !ok {
		s.log.Warn("email provider not found", "provider", providerName)
		return nil
	}

	// Parse events (Provider handles signature verification)
	events, err := p.ParseWebhook(ctx, nil, headers, body)
	if err != nil {
		s.log.Warn("failed to parse email webhook", "error", err, "provider", providerName)
		return nil
	}

	for _, event := range events {
		s.processEmailEvent(ctx, event)
	}

	return nil
}

func (s *WebhookService) processEmailEvent(ctx context.Context, event ports.WebhookEvent) {
	// 1. Idempotency Check
	if event.EmailProviderID != "" {
		exists, err := s.emailEventRepo.ExistsByProviderEventID(ctx, event.EmailProviderID)
		if err == nil && exists {
			s.log.Debug("ignoring duplicate email event", "provider_event_id", event.EmailProviderID)
			return
		}
	}

	// 2. Lookup message to get tenantID and campaignID
	msg, err := s.messageRepo.GetByProviderID(ctx, event.ProviderMessageID)
	if err != nil {
		s.log.Warn("message not found for email event", "provider_msg_id", event.ProviderMessageID)
		return
	}

	// 3. Create Email Event Record
	emailEvent := &domain.EmailEvent{
		ID:              uuid.New(),
		TenantID:        msg.TenantID,
		MessageID:       msg.ID,
		CampaignID:      msg.CampaignID,
		EventType:       event.EmailEventType,
		Recipient:       event.Email,
		Timestamp:       event.Timestamp,
		ProviderEventID: event.EmailProviderID,
		UserAgent:       event.UserAgent,
		IPAddress:       event.IPAddress,
		URL:             event.URL,
		CreatedAt:       time.Now(),
	}

	if event.BounceType != "" {
		emailEvent.BounceType = &event.BounceType
		emailEvent.BounceReason = &event.BounceReason
	}

	err = s.emailEventRepo.Create(ctx, emailEvent)
	if err != nil {
		s.log.Error("failed to create email event", "error", err)
	}

	// 4. Update Message Status (Never downgrade)
	if event.Status != nil {
		if getStatusRank(*event.Status) > getStatusRank(msg.Status) {
			err = s.messageRepo.UpdateStatusByProviderID(ctx, event.ProviderMessageID, *event.Status, event.Timestamp)
			if err != nil {
				s.log.Error("failed to update message status from email event", "error", err)
			}
		}
	}

	// 5. Auto-suppression Logic (Section 12.11)
	switch event.EmailEventType {
	case domain.EmailEventBounce:
		// hard_bounce event -> auto-add email to suppression list (reason: hard_bounce)
		if event.BounceType == "hard" {
			err = s.suppressionSvc.AddToSuppression(ctx, msg.TenantID, event.Email, domain.SuppressionHardBounce)
			if err != nil {
				s.log.Error("failed to auto-suppress email after hard bounce", "error", err, "email", event.Email)
			}
		} else if event.BounceType == "soft" {
			s.handleSoftBounce(ctx, msg.TenantID, event.Email)
		}

	case domain.EmailEventComplaint:
		// complaint event -> auto-add to suppression (reason: complaint) AND auto-add to unsubscribe (scope: global)
		err = s.suppressionSvc.AddToSuppression(ctx, msg.TenantID, event.Email, domain.SuppressionComplaint)
		if err != nil {
			s.log.Error("failed to auto-suppress email after complaint", "error", err, "email", event.Email)
		}
		err = s.unsubscribeSvc.Unsubscribe(ctx, msg.TenantID, event.Email, domain.UnsubscribeScopeGlobal, nil, "complaint")
		if err != nil {
			s.log.Error("failed to auto-unsubscribe email after complaint", "error", err, "email", event.Email)
		}

	case domain.EmailEventUnsubscribe:
		err = s.unsubscribeSvc.Unsubscribe(ctx, msg.TenantID, event.Email, domain.UnsubscribeScopeGlobal, msg.CampaignID, "link_click")
		if err != nil {
			s.log.Error("failed to auto-unsubscribe email", "error", err, "email", event.Email)
		}
	}
}

func (s *WebhookService) handleSoftBounce(ctx context.Context, tenantID uuid.UUID, email string) {
	// After 3 soft bounces to the same email within 72 hours, treat as hard bounce: add to suppression list.
	from := time.Now().Add(-72 * time.Hour)
	eventType := domain.EmailEventSoftBounce
	events, _, err := s.emailEventRepo.GetByCampaignID(ctx, tenantID, uuid.Nil, &eventType, 1, 10) // Simplified lookup
	if err != nil {
		s.log.Error("failed to lookup soft bounces", "error", err, "email", email)
		return
	}

	count := 0
	for _, e := range events {
		if e.Recipient == email && e.Timestamp.After(from) {
			count++
		}
	}

	if count >= 3 {
		err = s.suppressionSvc.AddToSuppression(ctx, tenantID, email, domain.SuppressionHardBounce)
		if err != nil {
			s.log.Error("failed to auto-suppress email after 3 soft bounces", "error", err, "email", email)
		}
	}
}
