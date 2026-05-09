package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
)

// EmailSender is the interface exposed to OTHER modules for sending emails.
// It is available via module.Services.EmailSender
type EmailSender interface {
	// SendTransactional sends a single transactional email (password reset, OTP, invoice, etc.)
	// Uses the platform's default email provider (from env) unless tenant has own config.
	// Transactional emails SKIP unsubscribe checks (but still check suppression list).
	SendTransactional(ctx context.Context, tenantID uuid.UUID, req TransactionalEmailRequest) (*domain.Message, error)

	// SendWithTemplate renders an email template and sends it.
	SendWithTemplate(ctx context.Context, tenantID uuid.UUID, req TemplateEmailRequest) (*domain.Message, error)
}

type TransactionalEmailRequest struct {
	To          string // recipient email
	Subject     string
	HTMLBody    string                   // raw HTML
	TextBody    string                   // plain text fallback (optional)
	FromEmail   string                   // override default FROM (optional)
	FromName    string                   // override default FROM name (optional)
	ReplyTo     string                   // optional
	CC          []string                 // optional
	BCC         []string                 // optional
	Headers     map[string]string        // custom headers (optional)
	Attachments []domain.EmailAttachment // optional
	Tags        []string                 // for analytics grouping (e.g. "password_reset", "invoice")
	Metadata    map[string]string        // custom tracking data
}

type TemplateEmailRequest struct {
	To           string
	TemplateName string            // name of EmailTemplate
	Variables    map[string]string // variable substitutions: {"name": "Dinesh", "order_id": "ORD-123"}
	FromEmail    string            // optional override
	FromName     string            // optional override
	ReplyTo      string            // optional
	CC           []string
	BCC          []string
	Attachments  []domain.EmailAttachment
	Tags         []string
	Metadata     map[string]string
}
