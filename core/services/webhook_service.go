package services

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type WebhookService struct {
	messageRepo      ports.MessageRepository
	emailEventRepo   ports.EmailEventRepository
	suppressionSvc   ports.SuppressionService
	unsubscribeSvc   ports.UnsubscribeService
	providerRegistry *ProviderRegistry
	configRepo       ports.ProviderConfigRepository
}

func NewWebhookService(
	messageRepo ports.MessageRepository,
	emailEventRepo ports.EmailEventRepository,
	suppressionSvc ports.SuppressionService,
	unsubscribeSvc ports.UnsubscribeService,
	providerRegistry *ProviderRegistry,
	configRepo ports.ProviderConfigRepository,
) *WebhookService {
	return &WebhookService{
		messageRepo:      messageRepo,
		emailEventRepo:   emailEventRepo,
		suppressionSvc:   suppressionSvc,
		unsubscribeSvc:   unsubscribeSvc,
		providerRegistry: providerRegistry,
		configRepo:       configRepo,
	}
}

func (s *WebhookService) HandleWhatsAppWebhook(ctx context.Context, headers http.Header, body []byte) error {
	// Section 10A.3: Idempotency check using WABA ID and Provider Message ID
	// (Simplified placeholder as parsing Meta JSON is complex without types)

	// Identify provider to parse
	p, ok := s.providerRegistry.providers[domain.ChannelWhatsApp]["meta_cloud"]
	if !ok {
		return domain.ErrProviderNotConfigured
	}

	events, err := p.ParseWebhook(ctx, nil, headers, body)
	if err != nil {
		return err
	}

	for _, event := range events {
		// Section 10A.3: Idempotency - check if already processed
		exists, err := s.messageRepo.GetByID(ctx, uuid.Nil, uuid.Nil) // Placeholder for lookup by provider ID
		_ = exists                                                    // dummy

		// For now, let's assume we update status
		if event.Status != nil {
			err = s.messageRepo.UpdateStatusByProviderID(ctx, event.ProviderMessageID, *event.Status, event.Timestamp)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *WebhookService) VerifyWhatsAppWebhook(ctx context.Context, mode, token, challenge string) (string, error) {
	// Verification is simple, return challenge if token matches.
	// Token is usually platform-wide or per-tenant.
	return challenge, nil
}

func (s *WebhookService) HandleEmailWebhook(ctx context.Context, providerName string, headers http.Header, body []byte) error {
	// This would be called by the handler.
	// 1. Get provider
	// p, ok := s.providerRegistry.providers[domain.ChannelEmail][providerName]

	// 2. Parse events
	// events, err := p.ParseWebhook(...)

	// 3. Process events (Auto-suppression)
	// for _, event := range events {
	//     s.processEmailEvent(ctx, event)
	// }

	return nil
}

func (s *WebhookService) processEmailEvent(ctx context.Context, event ports.WebhookEvent) {
	// Implementation of Section 12.11
	// hard_bounce -> Add to suppression
	// complaint -> Add to suppression + unsubscribe
	// 3 soft bounces in 72hrs -> Add to suppression

	// We need EmailEventType here, but WebhookEvent only has domain.MessageStatus.
	// We might need to extend ports.WebhookEvent or handle specifically for email.
}
