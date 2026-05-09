package whatsapp

import (
	"context"
	"errors"
	"net/http"

	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type whatsappProvider struct{}

func NewWhatsAppProvider() ports.Provider {
	return &whatsappProvider{}
}

func (p *whatsappProvider) Channel() domain.Channel {
	return domain.ChannelWhatsApp
}

func (p *whatsappProvider) Send(ctx context.Context, cfg *domain.ProviderConfig, req ports.ProviderSendRequest) (*ports.ProviderSendResult, error) {
	return nil, errors.New("whatsapp provider not fully implemented")
}

func (p *whatsappProvider) SubmitTemplate(ctx context.Context, cfg *domain.ProviderConfig, tmpl domain.Template) (string, error) {
	return "", errors.New("whatsapp provider not fully implemented")
}

func (p *whatsappProvider) GetTemplateStatus(ctx context.Context, cfg *domain.ProviderConfig, providerTmplID string) (domain.TemplateStatus, *string, error) {
	return domain.TemplateStatusPending, nil, nil
}

func (p *whatsappProvider) ParseWebhook(ctx context.Context, cfg *domain.ProviderConfig, headers http.Header, body []byte) ([]ports.WebhookEvent, error) {
	return nil, nil
}

func (p *whatsappProvider) ValidateWebhookSignature(cfg *domain.ProviderConfig, headers http.Header, body []byte) bool {
	return true
}
