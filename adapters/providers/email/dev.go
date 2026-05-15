package email

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type devEmailProvider struct {
	outboxDir string
}

type devEmailRecord struct {
	ID        string            `json:"id"`
	To        string            `json:"to"`
	CC        []string          `json:"cc,omitempty"`
	BCC       []string          `json:"bcc,omitempty"`
	FromEmail string            `json:"from_email,omitempty"`
	FromName  string            `json:"from_name,omitempty"`
	ReplyTo   string            `json:"reply_to,omitempty"`
	Subject   string            `json:"subject"`
	HTMLBody  string            `json:"html_body,omitempty"`
	TextBody  string            `json:"text_body,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

func NewDevEmailProvider() ports.EmailProvider {
	outboxDir := os.Getenv("MESSAGING_DEV_EMAIL_OUTBOX")
	if outboxDir == "" {
		outboxDir = filepath.Join(os.TempDir(), "spur-messaging-outbox")
	}
	return &devEmailProvider{outboxDir: outboxDir}
}

func (p *devEmailProvider) Channel() domain.Channel {
	return domain.ChannelEmail
}

func (p *devEmailProvider) Send(ctx context.Context, cfg *domain.ProviderConfig, req ports.ProviderSendRequest) (*ports.ProviderSendResult, error) {
	if req.Text == nil {
		return nil, fmt.Errorf("dev email body is required")
	}
	return p.SendEmail(ctx, cfg, ports.EmailSendRequest{
		To:       req.Recipient,
		Subject:  "Development email",
		HTMLBody: *req.Text,
	})
}

func (p *devEmailProvider) SubmitTemplate(ctx context.Context, cfg *domain.ProviderConfig, tmpl domain.Template) (string, error) {
	return "", fmt.Errorf("templates for email are handled internally, not via provider submission")
}

func (p *devEmailProvider) GetTemplateStatus(ctx context.Context, cfg *domain.ProviderConfig, providerTmplID string) (domain.TemplateStatus, *string, error) {
	return domain.TemplateStatusApproved, nil, nil
}

func (p *devEmailProvider) ParseWebhook(ctx context.Context, cfg *domain.ProviderConfig, headers http.Header, body []byte) ([]ports.WebhookEvent, error) {
	return nil, nil
}

func (p *devEmailProvider) ValidateWebhookSignature(cfg *domain.ProviderConfig, headers http.Header, body []byte) bool {
	return false
}

func (p *devEmailProvider) SendEmail(ctx context.Context, cfg *domain.ProviderConfig, req ports.EmailSendRequest) (*ports.ProviderSendResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.To) == "" {
		return nil, fmt.Errorf("recipient email is required")
	}
	if strings.TrimSpace(req.Subject) == "" {
		return nil, fmt.Errorf("email subject is required")
	}
	if req.HTMLBody == "" && req.TextBody == "" {
		return nil, fmt.Errorf("email body is required")
	}

	now := time.Now()
	id := uuid.NewString()
	record := devEmailRecord{
		ID:        id,
		To:        req.To,
		CC:        req.CC,
		BCC:       req.BCC,
		FromEmail: req.FromEmail,
		FromName:  req.FromName,
		ReplyTo:   req.ReplyTo,
		Subject:   req.Subject,
		HTMLBody:  req.HTMLBody,
		TextBody:  req.TextBody,
		Headers:   req.Headers,
		Metadata:  req.Metadata,
		CreatedAt: now,
	}
	if err := os.MkdirAll(p.outboxDir, 0o755); err != nil {
		return nil, fmt.Errorf("create dev email outbox: %w", err)
	}
	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal dev email record: %w", err)
	}
	path := filepath.Join(p.outboxDir, now.Format("20060102T150405.000000000")+"-"+id+".json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return nil, fmt.Errorf("write dev email record: %w", err)
	}

	return &ports.ProviderSendResult{
		ProviderMessageID: id,
		Status:            domain.MessageStatusSent,
		Timestamp:         now,
	}, nil
}

func (p *devEmailProvider) SendBatch(ctx context.Context, cfg *domain.ProviderConfig, reqs []ports.EmailSendRequest) ([]ports.ProviderSendResult, error) {
	results := make([]ports.ProviderSendResult, 0, len(reqs))
	for _, req := range reqs {
		result, err := p.SendEmail(ctx, cfg, req)
		if err != nil {
			return nil, err
		}
		results = append(results, *result)
	}
	return results, nil
}
