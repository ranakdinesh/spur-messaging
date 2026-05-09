package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type twilioProvider struct {
	httpClient *http.Client
}

func NewTwilioProvider() ports.Provider {
	return &twilioProvider{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *twilioProvider) Channel() domain.Channel {
	return domain.ChannelSMS
}

func (p *twilioProvider) Send(ctx context.Context, cfg *domain.ProviderConfig, req ports.ProviderSendRequest) (*ports.ProviderSendResult, error) {
	creds, err := p.getCredentials(cfg)
	if err != nil {
		return nil, err
	}

	// Twilio API for SMS
	// API: https://api.twilio.com/2010-04-01/Accounts/{sid}/Messages.json

	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", creds.AccountSID)

	data := url.Values{}
	data.Set("To", req.Recipient)
	data.Set("From", creds.FromNumber)

	if req.MessageType == domain.MessageTypeText && req.Text != nil {
		data.Set("Body", *req.Text)
	} else if req.MessageType == domain.MessageTypeTemplate && req.Text != nil {
		// For SMS, we often just send the rendered text
		data.Set("Body", *req.Text)
	} else {
		return nil, fmt.Errorf("unsupported message type for twilio: %s", req.MessageType)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create twilio request: %w", err)
	}

	httpReq.SetBasicAuth(creds.AccountSID, creds.AuthToken)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("twilio request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var twilioErr struct {
			Code     int    `json:"code"`
			Message  string `json:"message"`
			Status   int    `json:"status"`
			MoreInfo string `json:"more_info"`
		}
		json.NewDecoder(resp.Body).Decode(&twilioErr)
		return nil, fmt.Errorf("twilio error: %s (code: %d)", twilioErr.Message, twilioErr.Code)
	}

	var result struct {
		SID    string `json:"sid"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode twilio response: %w", err)
	}

	return &ports.ProviderSendResult{
		ProviderMessageID: result.SID,
		Status:            domain.MessageStatusSent,
		Timestamp:         time.Now(),
	}, nil
}

func (p *twilioProvider) SubmitTemplate(ctx context.Context, cfg *domain.ProviderConfig, tmpl domain.Template) (string, error) {
	return "", domain.ErrProviderError // SMS templates are not typically managed via Twilio API in this way
}

func (p *twilioProvider) GetTemplateStatus(ctx context.Context, cfg *domain.ProviderConfig, providerTmplID string) (domain.TemplateStatus, *string, error) {
	return domain.TemplateStatusApproved, nil, nil
}

func (p *twilioProvider) ParseWebhook(ctx context.Context, cfg *domain.ProviderConfig, headers http.Header, body []byte) ([]ports.WebhookEvent, error) {
	return ParseTwilioWebhook(ctx, cfg, headers, body)
}

func (p *twilioProvider) ValidateWebhookSignature(cfg *domain.ProviderConfig, headers http.Header, body []byte) bool {
	// Twilio signature verification requires the full URL and sorted parameters.
	// For now, we return true and will implement if needed in webhook.go
	return true
}

func (p *twilioProvider) getCredentials(cfg *domain.ProviderConfig) (*domain.SMSCredentials, error) {
	var creds domain.SMSCredentials
	if err := json.Unmarshal(cfg.Credentials, &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sms credentials: %w", err)
	}
	return &creds, nil
}
