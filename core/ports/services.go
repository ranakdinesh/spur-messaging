package ports

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
)

type MessageService interface {
	Send(ctx context.Context, tenantID uuid.UUID, req SendMessageRequest) (*domain.Message, error)
	SendBulk(ctx context.Context, tenantID uuid.UUID, reqs []SendMessageRequest) ([]domain.Message, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Message, error)
	List(ctx context.Context, tenantID uuid.UUID, filter MessageFilter) ([]domain.Message, int, error)
	Retry(ctx context.Context, tenantID, id uuid.UUID) (*domain.Message, error)
}

type ConversationService interface {
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Conversation, error)
	List(ctx context.Context, tenantID uuid.UUID, filter ConversationFilter) ([]domain.Conversation, int, error)
	Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateConversationRequest) (*domain.Conversation, error)
	AddNote(ctx context.Context, tenantID, id, authorID uuid.UUID, body string) (*domain.Conversation, error)
}

type UpdateConversationRequest struct {
	Status             *domain.ConversationStatus        `json:"status"`
	HandoffStatus      *domain.ConversationHandoffStatus `json:"handoff_status"`
	AssignedAgentID    *uuid.UUID                        `json:"assigned_agent_id"`
	AssignedTeam       *string                           `json:"assigned_team"`
	Priority           *domain.ConversationPriority      `json:"priority"`
	Tags               *[]string                         `json:"tags"`
	FirstResponseDueAt *time.Time                        `json:"first_response_due_at"`
	ResolutionDueAt    *time.Time                        `json:"resolution_due_at"`
}

type SendMessageRequest struct {
	Channel          domain.Channel
	Recipient        string
	MessageType      domain.MessageType
	TemplateName     *string
	TemplateLanguage *string
	TemplateParams   map[string]string
	Text             *string
	MediaURL         *string
	MediaType        *string
	IdempotencyKey   string
	Metadata         map[string]string

	// Per-request overrides (Tier 1) — all optional
	// If empty, falls back to tenant config (Tier 2) then platform default (Tier 3)
	FromEmail   string   // Email: override sender address
	FromName    string   // Email: override sender name
	ReplyTo     string   // Email: override reply-to
	TrackOpens  *bool    // Email: override open tracking (pointer = nil means "use default")
	TrackClicks *bool    // Email: override click tracking
	SenderID    string   // SMS: override sender ID
	CC          []string // Email only
	BCC         []string // Email only
}

type TemplateService interface {
	Create(ctx context.Context, tenantID uuid.UUID, req CreateTemplateRequest) (*domain.Template, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Template, error)
	GetByName(ctx context.Context, tenantID uuid.UUID, name, language string) (*domain.Template, error)
	List(ctx context.Context, tenantID uuid.UUID, channel *domain.Channel, status *domain.TemplateStatus, page, perPage int) ([]domain.Template, int, error)
	Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateTemplateRequest) (*domain.Template, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	SubmitForApproval(ctx context.Context, tenantID, id uuid.UUID) (*domain.Template, error)
	SyncStatus(ctx context.Context, tenantID, id uuid.UUID) (*domain.Template, error)
}

type CreateTemplateRequest struct {
	Channel    domain.Channel
	Name       string
	Language   string
	Category   domain.TemplateCategory
	Components []domain.TemplateComponent
}

type UpdateTemplateRequest struct {
	Category   *domain.TemplateCategory
	Components *[]domain.TemplateComponent
}

type CampaignService interface {
	Create(ctx context.Context, tenantID uuid.UUID, req CreateCampaignRequest) (*domain.Campaign, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Campaign, error)
	List(ctx context.Context, tenantID uuid.UUID, status *domain.CampaignStatus, page, perPage int) ([]domain.Campaign, int, error)
	Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateCampaignRequest) (*domain.Campaign, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	Execute(ctx context.Context, tenantID, id uuid.UUID) error
	Pause(ctx context.Context, tenantID, id uuid.UUID) error
	Resume(ctx context.Context, tenantID, id uuid.UUID) error
	GetStats(ctx context.Context, tenantID, id uuid.UUID) (*domain.CampaignStats, error)
}

type CreateCampaignRequest struct {
	Name           string
	Channel        domain.Channel
	TemplateID     uuid.UUID
	TemplateParams map[string]string
	SegmentID      *uuid.UUID
	ContactIDs     []uuid.UUID
	ScheduledAt    *time.Time
}

type UpdateCampaignRequest struct {
	Name           *string
	TemplateParams *map[string]string
	SegmentID      *uuid.UUID
	ContactIDs     *[]uuid.UUID
	ScheduledAt    *time.Time
}

type ContactService interface {
	Create(ctx context.Context, tenantID uuid.UUID, req CreateContactRequest) (*domain.Contact, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Contact, error)
	List(ctx context.Context, tenantID uuid.UUID, filter ContactFilter) ([]domain.Contact, int, error)
	Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateContactRequest) (*domain.Contact, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	BulkImport(ctx context.Context, tenantID uuid.UUID, contacts []CreateContactRequest) (BulkImportResult, error)
	OptIn(ctx context.Context, tenantID, id uuid.UUID, channel domain.Channel, consent ConsentEvidence) error
	ConfirmOptIn(ctx context.Context, tenantID, id uuid.UUID, channel domain.Channel, consent ConsentEvidence) error
	OptOut(ctx context.Context, tenantID, id uuid.UUID, channel domain.Channel, consent ConsentEvidence) error
	HandleInboundConsentKeyword(ctx context.Context, tenantID uuid.UUID, channel domain.Channel, recipient, text string, consent ConsentEvidence) (domain.ConsentKeywordAction, error)
	ListConsentRecords(ctx context.Context, tenantID, contactID uuid.UUID, page, perPage int) ([]domain.ConsentRecord, error)
}

type CreateContactRequest struct {
	Phone      *string           `json:"phone"`
	Email      *string           `json:"email"`
	Name       *string           `json:"name"`
	Attributes map[string]string `json:"attributes"`
	Tags       []string          `json:"tags"`
}

type UpdateContactRequest struct {
	Phone      *string
	Email      *string
	Name       *string
	Attributes *map[string]string
	Tags       *[]string
}

type ConsentEvidence struct {
	Source              string     `json:"source"`
	Purpose             string     `json:"purpose"`
	Proof               string     `json:"proof"`
	IPAddress           string     `json:"ip_address"`
	UserAgent           string     `json:"user_agent"`
	Brand               string     `json:"brand"`
	Keyword             string     `json:"keyword"`
	Locale              string     `json:"locale"`
	ExpiresAt           *time.Time `json:"expires_at"`
	ConfirmedAt         *time.Time `json:"confirmed_at"`
	DoubleOptInRequired bool       `json:"double_opt_in_required"`
}

type WebhookService interface {
	HandleWhatsAppWebhook(ctx context.Context, headers http.Header, body []byte) error
	VerifyWhatsAppWebhook(ctx context.Context, mode, token, challenge string) (string, error)
}

type TenantWebhookService interface {
	CreateEndpoint(ctx context.Context, tenantID uuid.UUID, req CreateWebhookEndpointRequest) (*domain.WebhookEndpoint, error)
	GetEndpoint(ctx context.Context, tenantID, id uuid.UUID) (*domain.WebhookEndpoint, error)
	ListEndpoints(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]domain.WebhookEndpoint, int, error)
	UpdateEndpoint(ctx context.Context, tenantID, id uuid.UUID, req UpdateWebhookEndpointRequest) (*domain.WebhookEndpoint, error)
	DeleteEndpoint(ctx context.Context, tenantID, id uuid.UUID) error
	ListDeliveries(ctx context.Context, tenantID uuid.UUID, webhookID *uuid.UUID, page, perPage int) ([]domain.WebhookDelivery, int, error)
	TestEndpoint(ctx context.Context, tenantID, id uuid.UUID) (*domain.WebhookDelivery, error)
	ReplayDelivery(ctx context.Context, tenantID, id uuid.UUID) (*domain.WebhookDelivery, error)
	DeliverEvent(ctx context.Context, tenantID uuid.UUID, eventType domain.WebhookEventType, payload json.RawMessage) ([]domain.WebhookDelivery, error)
	ProcessDueDeliveries(ctx context.Context, limit int) error
}

type CreateWebhookEndpointRequest struct {
	URL    string                    `json:"url"`
	Events []domain.WebhookEventType `json:"events"`
	Secret string                    `json:"secret"`
}

type UpdateWebhookEndpointRequest struct {
	URL      *string                    `json:"url"`
	Events   *[]domain.WebhookEventType `json:"events"`
	Secret   *string                    `json:"secret"`
	IsActive *bool                      `json:"is_active"`
}

type EmailTemplateService interface {
	Create(ctx context.Context, tenantID uuid.UUID, req CreateEmailTemplateRequest) (*domain.EmailTemplate, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.EmailTemplate, error)
	List(ctx context.Context, tenantID uuid.UUID, category *domain.EmailCategory, page, perPage int) ([]domain.EmailTemplate, int, error)
	Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateEmailTemplateRequest) (*domain.EmailTemplate, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	Preview(ctx context.Context, tenantID, id uuid.UUID, variables map[string]string) (*EmailPreview, error)
	Duplicate(ctx context.Context, tenantID, id uuid.UUID, newName string) (*domain.EmailTemplate, error)
}

type CreateEmailTemplateRequest struct {
	Name        string
	Subject     string
	PreviewText string
	HTMLBody    string
	TextBody    string // auto-generated from HTML if empty
	Category    domain.EmailCategory
	Variables   []string // e.g. ["name", "order_id", "amount"]
}

type UpdateEmailTemplateRequest struct {
	Subject     *string
	PreviewText *string
	HTMLBody    *string
	TextBody    *string
	Category    *domain.EmailCategory
	Variables   *[]string
	IsActive    *bool
}

type EmailPreview struct {
	Subject  string // rendered with variables
	HTMLBody string // rendered with variables
	TextBody string
}

type EmailAnalyticsService interface {
	GetOverview(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*domain.EmailStats, error)
	GetCampaignReport(ctx context.Context, tenantID, campaignID uuid.UUID) (*domain.EmailCampaignStats, error)
	GetDomainReputation(ctx context.Context, tenantID uuid.UUID) (*domain.DomainReputation, error)
	GetTopLinks(ctx context.Context, tenantID, campaignID uuid.UUID, limit int) ([]domain.LinkStats, error)
	GetBounceReport(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*domain.BounceReport, error)
	GetEngagementByHour(ctx context.Context, tenantID uuid.UUID, from, to time.Time) ([]domain.HourlyEngagement, error)
}

type SegmentService interface {
	Create(ctx context.Context, tenantID uuid.UUID, segment *domain.Segment) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Segment, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]domain.Segment, error)
	Update(ctx context.Context, tenantID, id uuid.UUID, segment *domain.Segment) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	ResolveContacts(ctx context.Context, tenantID, id uuid.UUID, page, perPage int) ([]domain.Contact, int, error)
}

type UnsubscribeService interface {
	Unsubscribe(ctx context.Context, tenantID uuid.UUID, email string, scope domain.UnsubscribeScope, campaignID *uuid.UUID, reason string) error
	Resubscribe(ctx context.Context, tenantID, id uuid.UUID) error
	IsUnsubscribed(ctx context.Context, tenantID uuid.UUID, email string) (bool, error)
	List(ctx context.Context, tenantID uuid.UUID, scope *domain.UnsubscribeScope, page, perPage int) ([]domain.Unsubscribe, int, error)
	// HandleUnsubscribeWebhook is called from the public unsubscribe endpoint
	HandleUnsubscribeLink(ctx context.Context, token string) error
}

type BulkImportResult struct {
	Imported   int              `json:"imported"`
	Duplicates int              `json:"duplicates"`
	Errors     []ImportRowError `json:"errors"`
	Total      int              `json:"total"`
}

type ImportRowError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

type SuppressionService interface {
	IsSuppressed(ctx context.Context, tenantID uuid.UUID, channel domain.Channel, recipient string) (bool, error)
	AddToSuppression(ctx context.Context, tenantID uuid.UUID, channel domain.Channel, recipient string, reason domain.SuppressionReason) error
	RemoveFromSuppression(ctx context.Context, tenantID, id uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, reason *domain.SuppressionReason, page, perPage int) ([]domain.SuppressionEntry, int, error)
	BulkCheck(ctx context.Context, tenantID uuid.UUID, channel domain.Channel, recipients []string) ([]string, error)
}
