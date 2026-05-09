package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type postmarkProvider struct {
	client *http.Client
}

func NewPostmarkProvider() ports.EmailProvider {
	return &postmarkProvider{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *postmarkProvider) Channel() domain.Channel {
	return domain.ChannelEmail
}

func (p *postmarkProvider) Send(ctx context.Context, cfg *domain.ProviderConfig, req ports.ProviderSendRequest) (*ports.ProviderSendResult, error) {
	return nil, fmt.Errorf("use SendEmail for email channel")
}

func (p *postmarkProvider) SubmitTemplate(ctx context.Context, cfg *domain.ProviderConfig, tmpl domain.Template) (string, error) {
	return "", fmt.Errorf("templates for email are handled internally")
}

func (p *postmarkProvider) GetTemplateStatus(ctx context.Context, cfg *domain.ProviderConfig, providerTmplID string) (domain.TemplateStatus, *string, error) {
	return domain.TemplateStatusApproved, nil, nil
}

func (p *postmarkProvider) SendEmail(ctx context.Context, cfg *domain.ProviderConfig, req ports.EmailSendRequest) (*ports.ProviderSendResult, error) {
	creds, err := p.getCredentials(cfg)
	if err != nil {
		return nil, err
	}

	pmReq := p.buildPostmarkRequest(req)
	body, err := json.Marshal(pmReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.postmarkapp.com/email", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("X-Postmark-Server-Token", creds.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var pmErr PostmarkResponse
		_ = json.NewDecoder(resp.Body).Decode(&pmErr)
		return nil, fmt.Errorf("postmark error: %s (code: %d)", pmErr.Message, pmErr.ErrorCode)
	}

	var pmRes PostmarkResponse
	if err := json.NewDecoder(resp.Body).Decode(&pmRes); err != nil {
		return nil, err
	}

	return &ports.ProviderSendResult{
		ProviderMessageID: pmRes.MessageID,
		Status:            domain.MessageStatusSent,
		Timestamp:         time.Now(),
	}, nil
}

func (p *postmarkProvider) SendBatch(ctx context.Context, cfg *domain.ProviderConfig, reqs []ports.EmailSendRequest) ([]ports.ProviderSendResult, error) {
	results := make([]ports.ProviderSendResult, 0, len(reqs))
	for _, req := range reqs {
		res, err := p.SendEmail(ctx, cfg, req)
		if err != nil {
			return nil, err
		}
		results = append(results, *res)
	}
	return results, nil
}

func (p *postmarkProvider) ParseWebhook(ctx context.Context, cfg *domain.ProviderConfig, headers http.Header, body []byte) ([]ports.WebhookEvent, error) {
	return ParsePostmarkWebhook(ctx, cfg, headers, body)
}

func (p *postmarkProvider) ValidateWebhookSignature(cfg *domain.ProviderConfig, headers http.Header, body []byte) bool {
	// Postmark doesn't have a standard signing mechanism, usually relies on shared secret in URL or IP whitelist.
	return true
}

func (p *postmarkProvider) getCredentials(cfg *domain.ProviderConfig) (domain.EmailCredentials, error) {
	var creds domain.EmailCredentials
	if err := json.Unmarshal(cfg.Credentials, &creds); err != nil {
		return creds, fmt.Errorf("unmarshal credentials: %w", err)
	}
	return creds, nil
}

func (p *postmarkProvider) buildPostmarkRequest(req ports.EmailSendRequest) map[string]any {
	pmReq := map[string]any{
		"From":     fmt.Sprintf("%s <%s>", req.FromName, req.FromEmail),
		"To":       req.To,
		"Subject":  req.Subject,
		"HtmlBody": req.HTMLBody,
		"TextBody": req.TextBody,
	}

	if req.ReplyTo != "" {
		pmReq["ReplyTo"] = req.ReplyTo
	}

	if len(req.CC) > 0 {
		pmReq["Cc"] = strings.Join(req.CC, ",")
	}
	if len(req.BCC) > 0 {
		pmReq["Bcc"] = strings.Join(req.BCC, ",")
	}

	if req.TrackOpens {
		pmReq["TrackOpens"] = true
	}
	if req.TrackClicks {
		pmReq["TrackLinks"] = "HtmlAndText"
	}

	if len(req.Metadata) > 0 {
		pmReq["Metadata"] = req.Metadata
	}

	if len(req.Tags) > 0 {
		pmReq["Tag"] = req.Tags[0] // Postmark supports one tag per message
	}

	if len(req.Headers) > 0 {
		headers := make([]map[string]string, 0, len(req.Headers))
		for k, v := range req.Headers {
			headers = append(headers, map[string]string{"Name": k, "Value": v})
		}
		pmReq["Headers"] = headers
	}

	if len(req.Attachments) > 0 {
		attachments := make([]map[string]any, len(req.Attachments))
		for i, att := range req.Attachments {
			attachments[i] = map[string]any{
				"Name":        att.Filename,
				"Content":     att.Content,
				"ContentType": att.ContentType,
			}
			if att.ContentID != "" {
				attachments[i]["ContentID"] = att.ContentID
			}
		}
		pmReq["Attachments"] = attachments
	}

	return pmReq
}
