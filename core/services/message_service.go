package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type MessageService struct {
	repo             ports.MessageRepository
	contactRepo      ports.ContactRepository
	templateRepo     ports.TemplateRepository
	queue            ports.MessageQueue
	suppressionSvc   ports.SuppressionService
	unsubscribeSvc   ports.UnsubscribeService
	providerRegistry *ProviderRegistry
}

func NewMessageService(
	repo ports.MessageRepository,
	contactRepo ports.ContactRepository,
	templateRepo ports.TemplateRepository,
	queue ports.MessageQueue,
	suppressionSvc ports.SuppressionService,
	unsubscribeSvc ports.UnsubscribeService,
	providerRegistry *ProviderRegistry,
) *MessageService {
	return &MessageService{
		repo:             repo,
		contactRepo:      contactRepo,
		templateRepo:     templateRepo,
		queue:            queue,
		suppressionSvc:   suppressionSvc,
		unsubscribeSvc:   unsubscribeSvc,
		providerRegistry: providerRegistry,
	}
}

func (s *MessageService) Send(ctx context.Context, tenantID uuid.UUID, req ports.SendMessageRequest) (*domain.Message, error) {
	// 1. Resolve provider config (Section 10A.2)
	_, _, err := s.providerRegistry.GetProvider(ctx, tenantID, req.Channel)
	if err != nil {
		return nil, domain.ErrProviderNotConfigured
	}

	// 2. Validate contact exists
	var contact *domain.Contact
	if req.Channel == domain.ChannelEmail {
		contact, err = s.contactRepo.GetByEmail(ctx, tenantID, req.Recipient)
	} else {
		contact, err = s.contactRepo.GetByPhone(ctx, tenantID, req.Recipient)
	}
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewNotFoundError("contact")
		}
		return nil, err
	}

	// 3. Check opt-in (Section 10A.2)
	var optedIn bool
	switch req.Channel {
	case domain.ChannelWhatsApp:
		optedIn = contact.OptInWhatsApp == domain.OptInStatusOptedIn
	case domain.ChannelSMS:
		optedIn = contact.OptInSMS == domain.OptInStatusOptedIn
	case domain.ChannelEmail:
		optedIn = contact.OptInEmail == domain.OptInStatusOptedIn
	}

	if !optedIn {
		return nil, domain.ErrOptInRequired
	}

	// 4. Rate limit check (Section 10A.2 - placeholder)

	// 5. Channel-specific checks
	if req.Channel == domain.ChannelWhatsApp {
		if req.MessageType == domain.MessageTypeTemplate {
			if req.TemplateName == nil {
				return nil, domain.NewValidationError("template_name", "template name is required for template messages")
			}
			lang := "en"
			if req.TemplateLanguage != nil && *req.TemplateLanguage != "" {
				lang = *req.TemplateLanguage
			}
			tmpl, err := s.templateRepo.GetByName(ctx, tenantID, *req.TemplateName, lang)
			if err != nil {
				return nil, domain.NewNotFoundError("template")
			}
			if tmpl.Status != domain.TemplateStatusApproved {
				return nil, domain.ErrTemplateNotApproved
			}
		} else {
			// Section 10A.2: Within 24hr session window?
			// Placeholder: check conversation history
		}
	}

	if req.Channel == domain.ChannelEmail {
		// Section 10A.2: Check suppression
		suppressed, err := s.suppressionSvc.IsSuppressed(ctx, tenantID, req.Recipient)
		if err != nil {
			return nil, err
		}
		if suppressed {
			return nil, domain.ErrSuppressed
		}

		// Section 10A.2: Check unsubscribe
		unsubscribed, err := s.unsubscribeSvc.IsUnsubscribed(ctx, tenantID, req.Recipient)
		if err != nil {
			return nil, err
		}
		if unsubscribed {
			return nil, domain.ErrUnsubscribed
		}
	}

	// 6. Create message
	msg := &domain.Message{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Channel:        req.Channel,
		Direction:      "outbound",
		Recipient:      req.Recipient,
		MessageType:    req.MessageType,
		TemplateName:   req.TemplateName,
		TemplateParams: req.TemplateParams,
		TextBody:       req.Text,
		MediaURL:       req.MediaURL,
		MediaType:      req.MediaType,
		Status:         domain.MessageStatusQueued,
		CreatedAt:      time.Now(),
		Metadata:       req.Metadata,
	}

	if req.TemplateName != nil {
		lang := "en"
		if req.TemplateLanguage != nil && *req.TemplateLanguage != "" {
			lang = *req.TemplateLanguage
		}
		tmpl, err := s.templateRepo.GetByName(ctx, tenantID, *req.TemplateName, lang)
		if err == nil {
			msg.TemplateID = &tmpl.ID
		}
	}

	err = s.repo.Create(ctx, msg)
	if err != nil {
		return nil, err
	}

	// 7. Enqueue
	priority := 0
	if req.Metadata != nil && req.Metadata["priority"] == "high" {
		priority = 1
	}

	err = s.queue.Enqueue(ctx, ports.QueueMessage{
		MessageID: msg.ID,
		TenantID:  tenantID,
		Channel:   req.Channel,
		Priority:  priority,
	})
	if err != nil {
		return nil, domain.ErrQueueUnavailable
	}

	return msg, nil
}

func (s *MessageService) SendBulk(ctx context.Context, tenantID uuid.UUID, reqs []ports.SendMessageRequest) ([]domain.Message, error) {
	var msgs []domain.Message
	for _, req := range reqs {
		msg, err := s.Send(ctx, tenantID, req)
		if err != nil {
			// In bulk, we might want to continue or return first error
			return nil, err
		}
		msgs = append(msgs, *msg)
	}
	return msgs, nil
}

func (s *MessageService) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Message, error) {
	return s.repo.GetByID(ctx, tenantID, id)
}

func (s *MessageService) List(ctx context.Context, tenantID uuid.UUID, filter ports.MessageFilter) ([]domain.Message, int, error) {
	return s.repo.List(ctx, tenantID, filter)
}

func (s *MessageService) Retry(ctx context.Context, tenantID, id uuid.UUID) (*domain.Message, error) {
	msg, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if msg.Status != domain.MessageStatusFailed {
		return nil, errors.New("only failed messages can be retried")
	}

	msg.Status = domain.MessageStatusQueued
	// Update status in repo...
	// (Needs UpdateStatus method in repo)

	err = s.queue.Enqueue(ctx, ports.QueueMessage{
		MessageID: msg.ID,
		TenantID:  tenantID,
		Channel:   msg.Channel,
		Priority:  0,
	})
	if err != nil {
		return nil, err
	}

	return msg, nil
}
