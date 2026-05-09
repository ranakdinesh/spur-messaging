package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type EmailSender struct {
	messageSvc ports.MessageService
}

func NewEmailSender(messageSvc ports.MessageService) *EmailSender {
	return &EmailSender{
		messageSvc: messageSvc,
	}
}

func (s *EmailSender) SendTransactional(ctx context.Context, tenantID uuid.UUID, req ports.TransactionalEmailRequest) (*domain.Message, error) {
	sendReq := ports.SendMessageRequest{
		Channel:     domain.ChannelEmail,
		Recipient:   req.To,
		MessageType: domain.MessageTypeText,
		Text:        &req.HTMLBody,
		FromEmail:   req.FromEmail,
		FromName:    req.FromName,
		ReplyTo:     req.ReplyTo,
		CC:          req.CC,
		BCC:         req.BCC,
		Metadata:    req.Metadata,
	}
	if sendReq.Metadata == nil {
		sendReq.Metadata = make(map[string]string)
	}
	sendReq.Metadata["category"] = "transactional"
	sendReq.Metadata["subject"] = req.Subject

	return s.messageSvc.Send(ctx, tenantID, sendReq)
}

func (s *EmailSender) SendWithTemplate(ctx context.Context, tenantID uuid.UUID, req ports.TemplateEmailRequest) (*domain.Message, error) {
	sendReq := ports.SendMessageRequest{
		Channel:        domain.ChannelEmail,
		Recipient:      req.To,
		MessageType:    domain.MessageTypeTemplate,
		TemplateName:   &req.TemplateName,
		TemplateParams: req.Variables,
		FromEmail:      req.FromEmail,
		FromName:       req.FromName,
		ReplyTo:        req.ReplyTo,
		CC:             req.CC,
		BCC:            req.BCC,
		Metadata:       req.Metadata,
	}
	if sendReq.Metadata == nil {
		sendReq.Metadata = make(map[string]string)
	}
	sendReq.Metadata["category"] = "transactional"

	return s.messageSvc.Send(ctx, tenantID, sendReq)
}
