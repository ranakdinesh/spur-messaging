package ports

import (
	"context"
	"net/http"
	"time"

	"github.com/ranakdinesh/spur-messaging/core/domain"
)

// Provider is implemented by each channel adapter (whatsapp, sms, email)
type Provider interface {
	Channel() domain.Channel
	Send(ctx context.Context, cfg *domain.ProviderConfig, req ProviderSendRequest) (*ProviderSendResult, error)
	SubmitTemplate(ctx context.Context, cfg *domain.ProviderConfig, tmpl domain.Template) (string, error)
	GetTemplateStatus(ctx context.Context, cfg *domain.ProviderConfig, providerTmplID string) (domain.TemplateStatus, *string, error)
	ParseWebhook(ctx context.Context, cfg *domain.ProviderConfig, headers http.Header, body []byte) ([]WebhookEvent, error)
	ValidateWebhookSignature(cfg *domain.ProviderConfig, headers http.Header, body []byte) bool
}

type ProviderSendRequest struct {
	Recipient          string
	MessageType        domain.MessageType
	TemplateName       *string
	TemplateLanguage   *string
	TemplateParams     map[string]string
	TemplateComponents []domain.TemplateComponent
	Text               *string
	MediaURL           *string
	MediaType          *string
	ReplyToMsgID       *string
}

type ProviderSendResult struct {
	ProviderMessageID string
	Status            domain.MessageStatus
	Cost              *float64
	Timestamp         time.Time
}

type WebhookEventType string

const (
	WebhookEventStatusUpdate WebhookEventType = "status_update" // delivery receipt
	WebhookEventIncoming     WebhookEventType = "incoming"      // inbound message
)

type WebhookEvent struct {
	Type              WebhookEventType
	ProviderMessageID string
	Status            *domain.MessageStatus // for status updates
	Timestamp         time.Time
	From              *string // for incoming messages
	Text              *string
	MediaURL          *string
	WABAID            string // to route to correct tenant

	// Email-specific fields
	EmailEventType  domain.EmailEventType
	EmailProviderID string
	Email           string
	URL             string // For click events
	BounceType      string
	BounceReason    string
	UserAgent       string
	IPAddress       string
}

// EmailProvider extends Provider with email-specific capabilities.
// Implemented by sendgrid.go, mailgun.go, postmark.go
type EmailProvider interface {
	Provider // inherits Send, ParseWebhook, ValidateWebhookSignature

	// SendEmail sends a fully formed email with all email-specific fields.
	// This is called by the email service instead of the generic Send().
	SendEmail(ctx context.Context, cfg *domain.ProviderConfig, req EmailSendRequest) (*ProviderSendResult, error)

	// SendBatch sends multiple emails in a single API call (if provider supports it).
	// SendGrid supports up to 1000 per batch. Mailgun/Postmark have their own limits.
	SendBatch(ctx context.Context, cfg *domain.ProviderConfig, reqs []EmailSendRequest) ([]ProviderSendResult, error)
}

type EmailSendRequest struct {
	To          string
	CC          []string
	BCC         []string
	FromEmail   string
	FromName    string
	ReplyTo     string
	Subject     string
	HTMLBody    string
	TextBody    string
	Headers     map[string]string // List-Unsubscribe, X-Custom-Header, etc.
	Attachments []domain.EmailAttachment
	Tags        []string             // provider tags for filtering/analytics
	Category    domain.EmailCategory // affects sending priority
	TrackOpens  bool
	TrackClicks bool
	Metadata    map[string]string // custom args passed to provider (returned in webhooks)
	IPPool      string            // SendGrid IP pool name (optional)
}
