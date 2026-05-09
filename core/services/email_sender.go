package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type EmailSender struct {
	messageRepo    ports.MessageRepository
	templateRepo   ports.EmailTemplateRepository
	suppressionSvc ports.SuppressionService
	queue          ports.MessageQueue
}

func NewEmailSender(
	messageRepo ports.MessageRepository,
	templateRepo ports.EmailTemplateRepository,
	suppressionSvc ports.SuppressionService,
	queue ports.MessageQueue,
) *EmailSender {
	return &EmailSender{
		messageRepo:    messageRepo,
		templateRepo:   templateRepo,
		suppressionSvc: suppressionSvc,
		queue:          queue,
	}
}

func (s *EmailSender) SendTransactional(ctx context.Context, tenantID uuid.UUID, req ports.TransactionalEmailRequest) (*domain.Message, error) {
	// 1. Check suppression list
	suppressed, err := s.suppressionSvc.IsSuppressed(ctx, tenantID, req.To)
	if err != nil {
		return nil, err
	}
	if suppressed {
		return nil, domain.ErrForbidden // Or a more specific error
	}

	// 2. Create message record
	msg := &domain.Message{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Channel:     domain.ChannelEmail,
		Direction:   "outbound",
		Recipient:   req.To,
		MessageType: domain.MessageTypeText,
		TextBody:    &req.HTMLBody, // Using HTMLBody as primary for email
		Status:      domain.MessageStatusQueued,
		CreatedAt:   time.Now(),
		Metadata:    req.Metadata,
	}

	if msg.Metadata == nil {
		msg.Metadata = make(map[string]string)
	}
	msg.Metadata["subject"] = req.Subject
	if req.FromEmail != "" {
		msg.Metadata["from_email"] = req.FromEmail
	}
	if req.FromName != "" {
		msg.Metadata["from_name"] = req.FromName
	}

	err = s.messageRepo.Create(ctx, msg)
	if err != nil {
		return nil, err
	}

	// 3. Enqueue
	err = s.queue.Enqueue(ctx, ports.QueueMessage{
		MessageID: msg.ID,
		TenantID:  tenantID,
		Channel:   domain.ChannelEmail,
		Priority:  1, // Transactional usually high priority
	})
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *EmailSender) SendWithTemplate(ctx context.Context, tenantID uuid.UUID, req ports.TemplateEmailRequest) (*domain.Message, error) {
	// 1. Resolve template
	_, err := s.templateRepo.GetByName(ctx, tenantID, req.TemplateName)
	if err != nil {
		return nil, err
	}

	// 2. Check suppression
	suppressed, err := s.suppressionSvc.IsSuppressed(ctx, tenantID, req.To)
	if err != nil {
		return nil, err
	}
	if suppressed {
		return nil, domain.ErrForbidden
	}

	// 3. Create message
	msg := &domain.Message{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Channel:        domain.ChannelEmail,
		Direction:      "outbound",
		Recipient:      req.To,
		MessageType:    domain.MessageTypeTemplate,
		TemplateName:   &req.TemplateName,
		TemplateParams: req.Variables,
		Status:         domain.MessageStatusQueued,
		CreatedAt:      time.Now(),
		Metadata:       req.Metadata,
	}

	err = s.messageRepo.Create(ctx, msg)
	if err != nil {
		return nil, err
	}

	// 4. Enqueue
	err = s.queue.Enqueue(ctx, ports.QueueMessage{
		MessageID: msg.ID,
		TenantID:  tenantID,
		Channel:   domain.ChannelEmail,
		Priority:  1,
	})
	if err != nil {
		return nil, err
	}

	return msg, nil
}
