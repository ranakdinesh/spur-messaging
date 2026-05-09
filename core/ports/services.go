package ports

import (
	"context"
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
	Metadata         map[string]string
}

type TemplateService interface {
	Create(ctx context.Context, tenantID uuid.UUID, req CreateTemplateRequest) (*domain.Template, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Template, error)
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
	BulkImport(ctx context.Context, tenantID uuid.UUID, contacts []CreateContactRequest) (int, error)
	OptIn(ctx context.Context, tenantID, id uuid.UUID, channel domain.Channel) error
	OptOut(ctx context.Context, tenantID, id uuid.UUID, channel domain.Channel) error
}

type CreateContactRequest struct {
	Phone      *string
	Email      *string
	Name       *string
	Attributes map[string]string
	Tags       []string
}

type UpdateContactRequest struct {
	Phone      *string
	Email      *string
	Name       *string
	Attributes *map[string]string
	Tags       *[]string
}

type WebhookService interface {
	HandleWhatsAppWebhook(ctx context.Context, headers http.Header, body []byte) error
	VerifyWhatsAppWebhook(ctx context.Context, mode, token, challenge string) (string, error)
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

type SuppressionService interface {
	IsSuppressed(ctx context.Context, tenantID uuid.UUID, email string) (bool, error)
	AddToSuppression(ctx context.Context, tenantID uuid.UUID, email string, reason domain.SuppressionReason) error
	RemoveFromSuppression(ctx context.Context, tenantID, id uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, reason *domain.SuppressionReason, page, perPage int) ([]domain.SuppressionEntry, int, error)
	BulkCheck(ctx context.Context, tenantID uuid.UUID, emails []string) ([]string, error)
}
