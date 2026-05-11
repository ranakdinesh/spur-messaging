package ports

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
)

type MessageRepository interface {
	Create(ctx context.Context, msg *domain.Message) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Message, error)
	GetByIdempotencyKey(ctx context.Context, tenantID uuid.UUID, key string) (*domain.Message, error)
	List(ctx context.Context, tenantID uuid.UUID, filter MessageFilter) ([]domain.Message, int, error)
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.MessageStatus, providerMsgID string) error
	UpdateStatusByProviderID(ctx context.Context, providerMsgID string, status domain.MessageStatus, timestamp time.Time) error
	GetByProviderID(ctx context.Context, providerMsgID string) (*domain.Message, error)
	GetByCampaignID(ctx context.Context, tenantID, campaignID uuid.UUID, page, perPage int) ([]domain.Message, int, error)
	ExistsForCampaign(ctx context.Context, campaignID uuid.UUID, recipient string) (bool, error)
	CountByCampaign(ctx context.Context, campaignID uuid.UUID) (int, error)
}

type ConversationRepository interface {
	GetActiveByRecipient(ctx context.Context, tenantID uuid.UUID, channel domain.Channel, recipient string, at time.Time) (*domain.Conversation, error)
	UpsertInbound(ctx context.Context, tenantID uuid.UUID, channel domain.Channel, recipient string, inboundAt time.Time) (*domain.Conversation, error)
	UpsertOutbound(ctx context.Context, tenantID uuid.UUID, channel domain.Channel, recipient string, outboundAt time.Time) (*domain.Conversation, error)
	GetConversationByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Conversation, error)
	ListConversations(ctx context.Context, tenantID uuid.UUID, filter ConversationFilter) ([]domain.Conversation, int, error)
	UpdateConversation(ctx context.Context, tenantID, id uuid.UUID, update ConversationUpdate) (*domain.Conversation, error)
	AddConversationNote(ctx context.Context, tenantID, id uuid.UUID, note domain.ConversationNote) (*domain.Conversation, error)
}

type ConversationFilter struct {
	Channel         *domain.Channel
	Status          *domain.ConversationStatus
	HandoffStatus   *domain.ConversationHandoffStatus
	AssignedAgentID *uuid.UUID
	Recipient       *string
	Tag             *string
	Page            int
	PerPage         int
}

type ConversationUpdate struct {
	Status             *domain.ConversationStatus
	HandoffStatus      *domain.ConversationHandoffStatus
	AssignedAgentID    *uuid.UUID
	AssignedTeam       *string
	Priority           *domain.ConversationPriority
	Tags               *[]string
	FirstResponseDueAt *time.Time
	ResolutionDueAt    *time.Time
}

type MessageFilter struct {
	Channel    *domain.Channel
	Status     *domain.MessageStatus
	Recipient  *string
	CampaignID *uuid.UUID
	DateFrom   *time.Time
	DateTo     *time.Time
	Page       int
	PerPage    int
}

type TemplateRepository interface {
	Create(ctx context.Context, tmpl *domain.Template) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Template, error)
	GetByName(ctx context.Context, tenantID uuid.UUID, name, language string) (*domain.Template, error)
	List(ctx context.Context, tenantID uuid.UUID, channel *domain.Channel, status *domain.TemplateStatus, page, perPage int) ([]domain.Template, int, error)
	Update(ctx context.Context, tmpl *domain.Template) error
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.TemplateStatus, providerID *string, rejectionReason *string) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

type ContactRepository interface {
	Create(ctx context.Context, contact *domain.Contact) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Contact, error)
	GetByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (*domain.Contact, error)
	GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.Contact, error)
	List(ctx context.Context, tenantID uuid.UUID, filter ContactFilter) ([]domain.Contact, int, error)
	Update(ctx context.Context, contact *domain.Contact) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	BulkCreate(ctx context.Context, contacts []domain.Contact) (int, error) // returns count created
	UpdateOptIn(ctx context.Context, tenantID, id uuid.UUID, channel domain.Channel, status domain.OptInStatus) error
	CreateConsentRecord(ctx context.Context, record *domain.ConsentRecord) error
	ListConsentRecords(ctx context.Context, tenantID, contactID uuid.UUID, page, perPage int) ([]domain.ConsentRecord, error)
	GetBySegment(ctx context.Context, tenantID, segmentID uuid.UUID, page, perPage int) ([]domain.Contact, int, error)
}

type ContactFilter struct {
	Phone   *string
	Email   *string
	Tag     *string
	OptedIn *domain.Channel // filter contacts opted in to this channel
	Page    int
	PerPage int
}

type CampaignRepository interface {
	Create(ctx context.Context, campaign *domain.Campaign) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Campaign, error)
	List(ctx context.Context, tenantID uuid.UUID, status *domain.CampaignStatus, page, perPage int) ([]domain.Campaign, int, error)
	Update(ctx context.Context, campaign *domain.Campaign) error
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.CampaignStatus) error
	UpdateStats(ctx context.Context, tenantID, id uuid.UUID, stats domain.CampaignStats) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	GetScheduledCampaigns(ctx context.Context, before time.Time) ([]domain.Campaign, error)
	GetRunningCampaigns(ctx context.Context) ([]domain.Campaign, error)
}

type ProviderConfigRepository interface {
	Create(ctx context.Context, cfg *domain.ProviderConfig) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.ProviderConfig, error)
	GetByChannel(ctx context.Context, tenantID uuid.UUID, channel domain.Channel) (*domain.ProviderConfig, error)
	GetByWABAID(ctx context.Context, wabaID string) (*domain.ProviderConfig, error) // for webhook routing
	List(ctx context.Context, tenantID uuid.UUID) ([]domain.ProviderConfig, error)
	Update(ctx context.Context, cfg *domain.ProviderConfig) error
	UpdateIsActive(ctx context.Context, tenantID, id uuid.UUID, isActive bool) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

type SegmentRepository interface {
	Create(ctx context.Context, segment *domain.Segment) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Segment, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]domain.Segment, error)
	Update(ctx context.Context, segment *domain.Segment) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	ResolveContacts(ctx context.Context, tenantID, segmentID uuid.UUID, page, perPage int) ([]domain.Contact, int, error)
}

type AnalyticsRepository interface {
	GetMessageStats(ctx context.Context, tenantID uuid.UUID, from, to time.Time, channel *domain.Channel) (*domain.MessageAnalytics, error)
	GetCampaignStats(ctx context.Context, tenantID, campaignID uuid.UUID) (*domain.CampaignStats, error)
}

type EmailTemplateRepository interface {
	Create(ctx context.Context, tmpl *domain.EmailTemplate) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.EmailTemplate, error)
	GetByName(ctx context.Context, tenantID uuid.UUID, name string) (*domain.EmailTemplate, error)
	List(ctx context.Context, tenantID uuid.UUID, category *domain.EmailCategory, page, perPage int) ([]domain.EmailTemplate, int, error)
	Update(ctx context.Context, tmpl *domain.EmailTemplate) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

type EmailEventRepository interface {
	Create(ctx context.Context, event *domain.EmailEvent) error
	CreateBatch(ctx context.Context, events []domain.EmailEvent) error
	GetByMessageID(ctx context.Context, tenantID, messageID uuid.UUID) ([]domain.EmailEvent, error)
	GetByCampaignID(ctx context.Context, tenantID, campaignID uuid.UUID, eventType *domain.EmailEventType, page, perPage int) ([]domain.EmailEvent, int, error)
	ExistsByProviderEventID(ctx context.Context, providerEventID string) (bool, error) // dedup
	GetStats(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*domain.EmailStats, error)
	GetCampaignStats(ctx context.Context, tenantID, campaignID uuid.UUID) (*domain.EmailCampaignStats, error)
}

type UnsubscribeRepository interface {
	Create(ctx context.Context, unsub *domain.Unsubscribe) error
	IsUnsubscribed(ctx context.Context, tenantID uuid.UUID, email string, scope domain.UnsubscribeScope, campaignID *uuid.UUID) (bool, error)
	List(ctx context.Context, tenantID uuid.UUID, scope *domain.UnsubscribeScope, page, perPage int) ([]domain.Unsubscribe, int, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error // re-subscribe
	GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) ([]domain.Unsubscribe, error)
}

type SuppressionRepository interface {
	Create(ctx context.Context, entry *domain.SuppressionEntry) error
	IsSuppressed(ctx context.Context, tenantID uuid.UUID, channel domain.Channel, recipient string) (bool, error)
	List(ctx context.Context, tenantID uuid.UUID, reason *domain.SuppressionReason, page, perPage int) ([]domain.SuppressionEntry, int, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error // remove from suppression (admin only)
	BulkCheck(ctx context.Context, tenantID uuid.UUID, channel domain.Channel, recipients []string) ([]string, error)
}

type WebhookRepository interface {
	CreateWebhookEndpoint(ctx context.Context, endpoint *domain.WebhookEndpoint) error
	GetWebhookEndpoint(ctx context.Context, tenantID, id uuid.UUID) (*domain.WebhookEndpoint, error)
	ListWebhookEndpoints(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]domain.WebhookEndpoint, int, error)
	UpdateWebhookEndpoint(ctx context.Context, endpoint *domain.WebhookEndpoint) error
	DeleteWebhookEndpoint(ctx context.Context, tenantID, id uuid.UUID) error
	CreateWebhookDelivery(ctx context.Context, delivery *domain.WebhookDelivery) error
	GetWebhookDelivery(ctx context.Context, tenantID, id uuid.UUID) (*domain.WebhookDelivery, error)
	ListWebhookDeliveries(ctx context.Context, tenantID uuid.UUID, webhookID *uuid.UUID, page, perPage int) ([]domain.WebhookDelivery, int, error)
	ListDueWebhookDeliveries(ctx context.Context, before time.Time, limit int) ([]domain.WebhookDelivery, error)
	UpdateWebhookDelivery(ctx context.Context, delivery *domain.WebhookDelivery) error
}

type WebhookEventPayload struct {
	EventID    uuid.UUID               `json:"event_id"`
	EventType  domain.WebhookEventType `json:"event_type"`
	TenantID   uuid.UUID               `json:"tenant_id"`
	Data       map[string]any          `json:"data"`
	OccurredAt time.Time               `json:"occurred_at"`
	Raw        json.RawMessage         `json:"-"`
}

type BillingRepository interface {
	CreateWalletLedgerEntry(ctx context.Context, entry *domain.WalletLedgerEntry) error
	ListWalletLedgerEntries(ctx context.Context, tenantID uuid.UUID, currency string, page, perPage int) ([]domain.WalletLedgerEntry, int, error)
	GetWalletBalance(ctx context.Context, tenantID uuid.UUID, currency string) (*domain.WalletBalance, error)
	WalletLedgerReferenceExists(ctx context.Context, tenantID uuid.UUID, referenceType string, referenceID uuid.UUID, entryType domain.WalletLedgerEntryType) (bool, error)
	GetActiveRateCard(ctx context.Context, tenantID uuid.UUID, channel domain.Channel, category, country, currency string, at time.Time) (*domain.RateCard, error)
	CreateRateCard(ctx context.Context, rate *domain.RateCard) error
}
