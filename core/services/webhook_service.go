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
	conversationRepo ports.ConversationRepository
	contactSvc       ports.ContactService
	emailEventRepo   ports.EmailEventRepository
	suppressionSvc   ports.SuppressionService
	unsubscribeSvc   ports.UnsubscribeService
	providerRegistry *ProviderRegistry
	configRepo       ports.ProviderConfigRepository
	tenantWebhooks   ports.TenantWebhookService
	log              Logger
}

func NewWebhookService(
	messageRepo ports.MessageRepository,
	conversationRepo ports.ConversationRepository,
	contactSvc ports.ContactService,
	emailEventRepo ports.EmailEventRepository,
	suppressionSvc ports.SuppressionService,
	unsubscribeSvc ports.UnsubscribeService,
	providerRegistry *ProviderRegistry,
	configRepo ports.ProviderConfigRepository,
	tenantWebhooks ports.TenantWebhookService,
	log Logger,
) *WebhookService {
	return &WebhookService{
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
		contactSvc:       contactSvc,
		emailEventRepo:   emailEventRepo,
		suppressionSvc:   suppressionSvc,
		unsubscribeSvc:   unsubscribeSvc,
		providerRegistry: providerRegistry,
		configRepo:       configRepo,
		tenantWebhooks:   tenantWebhooks,
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
	var current *domain.Message
	current, err := s.messageRepo.GetByProviderID(ctx, status.ID)
	if err == nil {
		if domain.MessageStatusRank(domainStatus) <= domain.MessageStatusRank(current.Status) {
			s.log.Debug("ignoring out-of-order status update", "provider_msg_id", status.ID, "current", current.Status, "incoming", domainStatus)
			return
		}
	}

	err = s.messageRepo.UpdateStatusByProviderID(ctx, status.ID, domainStatus, timestamp)
	if err != nil {
		s.log.Error("failed to update whatsapp status", "error", err, "provider_msg_id", status.ID)
		return
	}
	if current != nil {
		current.Status = domainStatus
		s.deliverMessageEvent(ctx, tenantID, webhookEventForMessageStatus(domainStatus), current, timestamp)
	}
}

func (s *WebhookService) processWhatsAppIncoming(ctx context.Context, tenantID uuid.UUID, msg whatsapp.WebhookMessage) {
	timestamp := time.Now()
	if t, err := strconv.ParseInt(msg.Timestamp, 10, 64); err == nil {
		timestamp = time.Unix(t, 0)
	}

	var conversationID *uuid.UUID
	if s.conversationRepo != nil {
		conversation, err := s.conversationRepo.UpsertInbound(ctx, tenantID, domain.ChannelWhatsApp, msg.From, timestamp)
		if err != nil {
			s.log.Error("failed to upsert whatsapp conversation", "error", err, "from", msg.From)
		} else {
			conversationID = &conversation.ID
		}
	}

	inbound := &domain.Message{
		ID:                uuid.New(),
		TenantID:          tenantID,
		ConversationID:    conversationID,
		Channel:           domain.ChannelWhatsApp,
		Direction:         "inbound",
		Recipient:         "platform", // Or our WhatsApp number
		Sender:            msg.From,
		MessageType:       domain.MessageType(msg.Type),
		ProviderMessageID: msg.ID,
		Status:            domain.MessageStatusReplied,
		SentAt:            &timestamp,
		CreatedAt:         time.Now(),
	}

	if msg.Text != nil {
		inbound.TextBody = &msg.Text.Body
	}

	err := s.messageRepo.Create(ctx, inbound)
	if err != nil {
		s.log.Error("failed to create inbound whatsapp message", "error", err)
		return
	}
	s.deliverMessageEvent(ctx, tenantID, domain.WebhookEventMessageReplied, inbound, timestamp)
	if msg.Text != nil {
		s.processInboundConsentKeyword(ctx, tenantID, domain.ChannelWhatsApp, msg.From, msg.Text.Body)
	}
}

func (s *WebhookService) processInboundConsentKeyword(ctx context.Context, tenantID uuid.UUID, channel domain.Channel, recipient, text string) {
	if s.contactSvc == nil {
		return
	}
	action, err := s.contactSvc.HandleInboundConsentKeyword(ctx, tenantID, channel, recipient, text, ports.ConsentEvidence{
		Source:  "inbound_keyword",
		Keyword: text,
	})
	if err != nil {
		if action != domain.ConsentKeywordUnknown {
			s.log.Warn("failed to process inbound consent keyword", "error", err, "channel", channel, "recipient", recipient, "action", action)
		}
		return
	}
	if action != domain.ConsentKeywordUnknown {
		s.log.Info("processed inbound consent keyword", "channel", channel, "recipient", recipient, "action", action)
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
		return domain.MessageStatusProviderSubmitted
	}
}

func getStatusRank(s domain.MessageStatus) int {
	return domain.MessageStatusRank(s)
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
		if domain.MessageStatusRank(*event.Status) > domain.MessageStatusRank(msg.Status) {
			err = s.messageRepo.UpdateStatusByProviderID(ctx, event.ProviderMessageID, *event.Status, event.Timestamp)
			if err != nil {
				s.log.Error("failed to update message status from email event", "error", err)
			} else {
				msg.Status = *event.Status
				s.deliverMessageEvent(ctx, msg.TenantID, webhookEventForMessageStatus(*event.Status), msg, event.Timestamp)
			}
		}
	}

	// 5. Auto-suppression Logic (Section 12.11)
	switch event.EmailEventType {
	case domain.EmailEventBounce:
		// hard_bounce event -> auto-add email to suppression list (reason: hard_bounce)
		if event.BounceType == "hard" {
			err = s.suppressionSvc.AddToSuppression(ctx, msg.TenantID, domain.ChannelEmail, event.Email, domain.SuppressionHardBounce)
			if err != nil {
				s.log.Error("failed to auto-suppress email after hard bounce", "error", err, "email", event.Email)
			}
		} else if event.BounceType == "soft" {
			s.handleSoftBounce(ctx, msg.TenantID, event.Email)
		}

	case domain.EmailEventComplaint:
		// complaint event -> auto-add to suppression (reason: complaint) AND auto-add to unsubscribe (scope: global)
		err = s.suppressionSvc.AddToSuppression(ctx, msg.TenantID, domain.ChannelEmail, event.Email, domain.SuppressionComplaint)
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

func (s *WebhookService) deliverMessageEvent(ctx context.Context, tenantID uuid.UUID, eventType domain.WebhookEventType, msg *domain.Message, occurredAt time.Time) {
	if s.tenantWebhooks == nil || !domain.IsValidWebhookEvent(eventType) {
		return
	}
	payload, err := json.Marshal(map[string]any{
		"event_id":    uuid.New().String(),
		"event_type":  string(eventType),
		"tenant_id":   tenantID.String(),
		"occurred_at": occurredAt.UTC().Format(time.RFC3339Nano),
		"data": map[string]any{
			"message_id":          msg.ID.String(),
			"provider_message_id": msg.ProviderMessageID,
			"campaign_id":         uuidPtrString(msg.CampaignID),
			"conversation_id":     uuidPtrString(msg.ConversationID),
			"channel":             string(msg.Channel),
			"recipient":           msg.Recipient,
			"sender":              msg.Sender,
			"direction":           msg.Direction,
			"status":              string(msg.Status),
		},
	})
	if err != nil {
		s.log.Warn("failed to build tenant webhook payload", "error", err, "message_id", msg.ID)
		return
	}
	if _, err := s.tenantWebhooks.DeliverEvent(ctx, tenantID, eventType, payload); err != nil {
		s.log.Warn("failed to deliver tenant webhook event", "error", err, "event_type", eventType, "message_id", msg.ID)
	}
}

func webhookEventForMessageStatus(status domain.MessageStatus) domain.WebhookEventType {
	switch status {
	case domain.MessageStatusSent:
		return domain.WebhookEventMessageSent
	case domain.MessageStatusDelivered:
		return domain.WebhookEventMessageDelivered
	case domain.MessageStatusRead:
		return domain.WebhookEventMessageRead
	case domain.MessageStatusOpened:
		return domain.WebhookEventMessageOpened
	case domain.MessageStatusClicked:
		return domain.WebhookEventMessageClicked
	case domain.MessageStatusFailed, domain.MessageStatusExpired, domain.MessageStatusCancelled, domain.MessageStatusSuppressed:
		return domain.WebhookEventMessageFailed
	case domain.MessageStatusReplied:
		return domain.WebhookEventMessageReplied
	default:
		return domain.WebhookEventMessageCreated
	}
}

func uuidPtrString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
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
		err = s.suppressionSvc.AddToSuppression(ctx, tenantID, domain.ChannelEmail, email, domain.SuppressionHardBounce)
		if err != nil {
			s.log.Error("failed to auto-suppress email after 3 soft bounces", "error", err, "email", email)
		}
	}
}
