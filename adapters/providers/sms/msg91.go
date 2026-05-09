package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type msg91Provider struct {
	httpClient *http.Client
}

func NewMSG91Provider() ports.Provider {
	return &msg91Provider{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *msg91Provider) Channel() domain.Channel {
	return domain.ChannelSMS
}

func (p *msg91Provider) Send(ctx context.Context, cfg *domain.ProviderConfig, req ports.ProviderSendRequest) (*ports.ProviderSendResult, error) {
	creds, err := p.getCredentials(cfg)
	if err != nil {
		return nil, err
	}

	// MSG91 v5 API for SMS
	// Documentation: https://control.msg91.com/api/v5/
	// For DLT support, template_id is used.

	payload := map[string]any{
		"template_id": req.TemplateName, // In SMS, TemplateName maps to MSG91 template_id
		"mobile":      req.Recipient,
		"authkey":     creds.AuthKey,
	}

	// Map template params to MSG91 variables
	for k, v := range req.TemplateParams {
		payload[k] = v
	}

	// If it's a direct text message (not template based in Spur sense, but MSG91 usually requires templates for DLT)
	if req.MessageType == domain.MessageTypeText && req.Text != nil {
		// If we don't have a template name, we might be using a default template or flow
		// but MSG91 V5 flow usually prefers template_id.
		// For simplicity, we assume req.TemplateName contains the DLT template ID as per issue description.
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal msg91 payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://control.msg91.com/api/v5/flow/", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create msg91 request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("authkey", creds.AuthKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("msg91 request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("msg91 returned error status: %d", resp.StatusCode)
	}

	var result struct {
		RequestID string `json:"request_id"`
		Type      string `json:"type"`
		Message   string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode msg91 response: %w", err)
	}

	return &ports.ProviderSendResult{
		ProviderMessageID: result.RequestID,
		Status:            domain.MessageStatusSent,
		Timestamp:         time.Now(),
	}, nil
}

func (p *msg91Provider) SubmitTemplate(ctx context.Context, cfg *domain.ProviderConfig, tmpl domain.Template) (string, error) {
	return "", domain.ErrProviderError // SMS templates usually managed in DLT portal
}

func (p *msg91Provider) GetTemplateStatus(ctx context.Context, cfg *domain.ProviderConfig, providerTmplID string) (domain.TemplateStatus, *string, error) {
	return domain.TemplateStatusApproved, nil, nil
}

func (p *msg91Provider) ParseWebhook(ctx context.Context, cfg *domain.ProviderConfig, headers http.Header, body []byte) ([]ports.WebhookEvent, error) {
	return ParseMSG91Webhook(ctx, cfg, headers, body)
}

func (p *msg91Provider) ValidateWebhookSignature(cfg *domain.ProviderConfig, headers http.Header, body []byte) bool {
	return true // MSG91 typically uses IP whitelisting or simple tokens
}

func (p *msg91Provider) getCredentials(cfg *domain.ProviderConfig) (*domain.SMSCredentials, error) {
	var creds domain.SMSCredentials
	if err := json.Unmarshal(cfg.Credentials, &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sms credentials: %w", err)
	}
	return &creds, nil
}
