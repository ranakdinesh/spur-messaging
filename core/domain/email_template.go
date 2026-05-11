package domain

import (
	"time"

	"github.com/google/uuid"
)

// EmailTemplate is a reusable HTML email template.
// SEPARATE from the WhatsApp Template entity — email templates have different
// structure (HTML body, subject, preview text) and don't need Meta approval.
type EmailTemplate struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	Name        string        // Unique per tenant: "welcome_email", "order_shipped"
	Subject     string        // Email subject line — supports {{variable}} substitution
	PreviewText string        // Preview text shown in inbox (first ~90 chars)
	HTMLBody    string        // Full HTML email body — supports {{variable}} substitution
	TextBody    string        // Plain text fallback (auto-generated from HTML if empty)
	Category    EmailCategory // transactional, marketing, notification
	Variables   []string      // List of variable names used in template: ["name", "order_id"]
	IsActive    bool
	Version     int // Auto-incremented on each update
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type EmailCategory string

const (
	EmailCategoryTransactional EmailCategory = "transactional"
	EmailCategoryMarketing     EmailCategory = "marketing"
	EmailCategoryNotification  EmailCategory = "notification"
)

// EmailMessage extends the base Message with email-specific fields.
// These fields are stored in the Message.Metadata JSONB column.
type EmailMessageMeta struct {
	Subject     string            `json:"subject"`
	FromEmail   string            `json:"from_email"`
	FromName    string            `json:"from_name"`
	ReplyTo     string            `json:"reply_to,omitempty"`
	CC          []string          `json:"cc,omitempty"`
	BCC         []string          `json:"bcc,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"` // Custom headers
	Attachments []EmailAttachment `json:"attachments,omitempty"`
	TrackOpens  bool              `json:"track_opens"`
	TrackClicks bool              `json:"track_clicks"`
	Tags        []string          `json:"tags,omitempty"` // Provider tags for filtering
	Category    EmailCategory     `json:"category"`
	IPPool      string            `json:"ip_pool,omitempty"` // SendGrid IP pool
}

type EmailAttachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`         // e.g. "application/pdf"
	Content     string `json:"content"`              // base64-encoded content
	ContentID   string `json:"content_id,omitempty"` // for inline images
}

// EmailEvent tracks granular email lifecycle events from provider webhooks.
// One message can have multiple events (sent → delivered → opened → clicked).
type EmailEvent struct {
	ID                uuid.UUID
	TenantID          uuid.UUID
	MessageID         uuid.UUID // FK to messaging.messages
	CampaignID        *uuid.UUID
	EventType         EmailEventType
	Recipient         string // email address
	Timestamp         time.Time
	ProviderEventID   string            // provider's unique event ID (for dedup)
	UserAgent         string            // for open/click events
	IPAddress         string            // for open/click events
	URL               string            // for click events: which link was clicked
	BounceType        *string           // "hard" or "soft" (for bounce events)
	BounceReason      *string           // SMTP error message
	ComplaintFeedback *string           // ISP complaint feedback type
	RawPayload        map[string]string // provider's raw webhook data for debugging
	CreatedAt         time.Time
}

type EmailEventType string

const (
	EmailEventDelivered   EmailEventType = "delivered"
	EmailEventBounce      EmailEventType = "bounce"
	EmailEventSoftBounce  EmailEventType = "soft_bounce"
	EmailEventOpen        EmailEventType = "open"
	EmailEventClick       EmailEventType = "click"
	EmailEventUnsubscribe EmailEventType = "unsubscribe"
	EmailEventComplaint   EmailEventType = "complaint" // ISP spam complaint
	EmailEventDropped     EmailEventType = "dropped"   // provider refused to send
	EmailEventDeferred    EmailEventType = "deferred"  // temp failure, provider retrying
)

// Unsubscribe tracks email opt-outs at multiple levels.
type Unsubscribe struct {
	ID         uuid.UUID
	TenantID   uuid.UUID
	Email      string
	Scope      UnsubscribeScope
	CampaignID *uuid.UUID // only if scope is "campaign"
	Reason     string     // "manual", "link_click", "complaint", "bounce"
	CreatedAt  time.Time
}

type UnsubscribeScope string

const (
	UnsubscribeScopeGlobal   UnsubscribeScope = "global"   // all emails from tenant
	UnsubscribeScopeCampaign UnsubscribeScope = "campaign" // specific campaign only
	UnsubscribeScopeCategory UnsubscribeScope = "category" // all marketing, keep transactional
)

// SuppressionEntry — addresses that must NEVER receive email.
// Hard bounces and complaints are auto-added. Cannot be overridden.
type SuppressionEntry struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Channel   Channel
	Recipient string
	Email     string
	Reason    SuppressionReason
	Source    string // "bounce_webhook", "complaint_webhook", "manual", "import"
	CreatedAt time.Time
}

type SuppressionReason string

const (
	SuppressionHardBounce SuppressionReason = "hard_bounce"
	SuppressionComplaint  SuppressionReason = "complaint"
	SuppressionManual     SuppressionReason = "manual"  // admin manually suppressed
	SuppressionInvalid    SuppressionReason = "invalid" // email validation failed
)
