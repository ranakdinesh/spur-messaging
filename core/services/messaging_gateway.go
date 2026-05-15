package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type MessagingGateway struct {
	messageSvc ports.MessageService
}

func NewMessagingGateway(messageSvc ports.MessageService) *MessagingGateway {
	return &MessagingGateway{messageSvc: messageSvc}
}

func (g *MessagingGateway) Submit(ctx context.Context, tenantID uuid.UUID, req ports.MessagingRequest) (*ports.MessagingReceipt, error) {
	sendReq := ports.SendMessageRequest{
		Channel:        req.Channel,
		Recipient:      req.Recipient,
		MessageType:    req.MessageType,
		TemplateParams: req.TemplateParams,
		IdempotencyKey: req.IdempotencyKey,
		FromEmail:      req.FromEmail,
		FromName:       req.FromName,
		ReplyTo:        req.ReplyTo,
		CC:             req.CC,
		BCC:            req.BCC,
		Metadata:       cloneMetadata(req.Metadata),
	}
	if sendReq.MessageType == "" {
		sendReq.MessageType = domain.MessageTypeText
	}
	if sendReq.Metadata == nil {
		sendReq.Metadata = make(map[string]string)
	}
	if req.Category != "" {
		sendReq.Metadata["category"] = req.Category
	}
	if req.Subject != "" {
		sendReq.Metadata["subject"] = req.Subject
	}
	if req.HTMLBody != "" {
		sendReq.Metadata["html_body"] = req.HTMLBody
	}
	if req.TextBody != "" {
		text := req.TextBody
		sendReq.Text = &text
		sendReq.Metadata["text_body"] = req.TextBody
	}
	if req.CorrelationID != "" {
		sendReq.Metadata["correlation_id"] = req.CorrelationID
	}
	if req.CallbackRef != "" {
		sendReq.Metadata["callback_ref"] = req.CallbackRef
	}
	if req.Priority != "" {
		sendReq.Metadata["priority"] = req.Priority
	}
	if req.TemplateName != "" {
		sendReq.TemplateName = &req.TemplateName
	}
	if req.TemplateLanguage != "" {
		sendReq.TemplateLanguage = &req.TemplateLanguage
	}
	if req.MediaURL != "" {
		sendReq.MediaURL = &req.MediaURL
	}
	if req.MediaType != "" {
		sendReq.MediaType = &req.MediaType
	}
	if req.Channel == domain.ChannelEmail && req.HTMLBody != "" {
		html := req.HTMLBody
		sendReq.Text = &html
	}

	msg, err := g.messageSvc.Send(ctx, tenantID, sendReq)
	if err != nil {
		return nil, err
	}

	return &ports.MessagingReceipt{
		MessageID:         msg.ID,
		TenantID:          msg.TenantID,
		Channel:           msg.Channel,
		Status:            msg.Status,
		Accepted:          msg.Status != domain.MessageStatusFailed && msg.Status != domain.MessageStatusSuppressed,
		IdempotencyKey:    req.IdempotencyKey,
		CorrelationID:     req.CorrelationID,
		ProviderMessageID: msg.ProviderMessageID,
		CreatedAt:         msg.CreatedAt,
	}, nil
}

func (g *MessagingGateway) GetResult(ctx context.Context, tenantID, messageID uuid.UUID) (*ports.MessagingResult, error) {
	msg, err := g.messageSvc.GetByID(ctx, tenantID, messageID)
	if err != nil {
		return nil, err
	}
	return &ports.MessagingResult{
		MessageID:         msg.ID,
		TenantID:          msg.TenantID,
		Channel:           msg.Channel,
		Status:            msg.Status,
		ProviderMessageID: msg.ProviderMessageID,
		ErrorCode:         stringValue(msg.ErrorCode),
		ErrorMessage:      stringValue(msg.ErrorMessage),
		SentAt:            msg.SentAt,
		DeliveredAt:       msg.DeliveredAt,
		ReadAt:            msg.ReadAt,
		FailedAt:          msg.FailedAt,
		Metadata:          msg.Metadata,
	}, nil
}

func cloneMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		return nil
	}
	clone := make(map[string]string, len(metadata))
	for key, value := range metadata {
		clone[key] = value
	}
	return clone
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
