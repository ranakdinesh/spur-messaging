package messaging

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/adapters/postgres"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

// Bridge adapters to map postgres.Store specific methods to generic ports.Repository interfaces

type messageRepoAdapter struct{ *postgres.Store }

type templateRepoAdapter struct{ *postgres.Store }

func (a templateRepoAdapter) Create(ctx context.Context, tmpl *domain.Template) error {
	return a.Store.CreateTemplate(ctx, tmpl)
}
func (a templateRepoAdapter) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Template, error) {
	return a.Store.GetTemplateByID(ctx, tenantID, id)
}
func (a templateRepoAdapter) GetByName(ctx context.Context, tenantID uuid.UUID, name, language string) (*domain.Template, error) {
	return a.Store.GetTemplateByName(ctx, tenantID, name, language)
}
func (a templateRepoAdapter) List(ctx context.Context, tenantID uuid.UUID, channel *domain.Channel, status *domain.TemplateStatus, page, perPage int) ([]domain.Template, int, error) {
	return a.Store.ListTemplates(ctx, tenantID, channel, status, page, perPage)
}
func (a templateRepoAdapter) Update(ctx context.Context, tmpl *domain.Template) error {
	return a.Store.UpdateTemplate(ctx, tmpl)
}
func (a templateRepoAdapter) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.TemplateStatus, providerID *string, rejectionReason *string) error {
	return a.Store.UpdateTemplateStatus(ctx, tenantID, id, status, providerID, rejectionReason)
}
func (a templateRepoAdapter) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return a.Store.DeleteTemplate(ctx, tenantID, id)
}

type contactRepoAdapter struct{ *postgres.Store }

func (a contactRepoAdapter) Create(ctx context.Context, contact *domain.Contact) error {
	return a.Store.CreateContact(ctx, contact)
}
func (a contactRepoAdapter) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Contact, error) {
	return a.Store.GetContactByID(ctx, tenantID, id)
}
func (a contactRepoAdapter) GetByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (*domain.Contact, error) {
	return a.Store.GetContactByPhone(ctx, tenantID, phone)
}
func (a contactRepoAdapter) GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.Contact, error) {
	return a.Store.GetContactByEmail(ctx, tenantID, email)
}
func (a contactRepoAdapter) List(ctx context.Context, tenantID uuid.UUID, filter ports.ContactFilter) ([]domain.Contact, int, error) {
	return a.Store.ListContacts(ctx, tenantID, filter)
}
func (a contactRepoAdapter) Update(ctx context.Context, contact *domain.Contact) error {
	return a.Store.UpdateContact(ctx, contact)
}
func (a contactRepoAdapter) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return a.Store.DeleteContact(ctx, tenantID, id)
}
func (a contactRepoAdapter) UpdateOptIn(ctx context.Context, tenantID, id uuid.UUID, channel domain.Channel, status domain.OptInStatus) error {
	return a.Store.UpdateOptIn(ctx, tenantID, id, channel, status)
}
func (a contactRepoAdapter) GetBySegment(ctx context.Context, tenantID, segmentID uuid.UUID, page, perPage int) ([]domain.Contact, int, error) {
	return a.Store.ResolveContacts(ctx, tenantID, segmentID, page, perPage)
}

type campaignRepoAdapter struct{ *postgres.Store }

func (a campaignRepoAdapter) Create(ctx context.Context, campaign *domain.Campaign) error {
	return a.Store.CreateCampaign(ctx, campaign)
}
func (a campaignRepoAdapter) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Campaign, error) {
	return a.Store.GetCampaignByID(ctx, tenantID, id)
}
func (a campaignRepoAdapter) List(ctx context.Context, tenantID uuid.UUID, status *domain.CampaignStatus, page, perPage int) ([]domain.Campaign, int, error) {
	return a.Store.ListCampaigns(ctx, tenantID, status, page, perPage)
}
func (a campaignRepoAdapter) Update(ctx context.Context, campaign *domain.Campaign) error {
	return a.Store.UpdateCampaign(ctx, campaign)
}
func (a campaignRepoAdapter) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.CampaignStatus) error {
	return a.Store.UpdateCampaignStatus(ctx, tenantID, id, status)
}
func (a campaignRepoAdapter) UpdateStats(ctx context.Context, tenantID, id uuid.UUID, stats domain.CampaignStats) error {
	return a.Store.UpdateCampaignStats(ctx, tenantID, id, stats)
}
func (a campaignRepoAdapter) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return a.Store.DeleteCampaign(ctx, tenantID, id)
}

type providerConfigRepoAdapter struct{ *postgres.Store }

func (a providerConfigRepoAdapter) Create(ctx context.Context, cfg *domain.ProviderConfig) error {
	return a.Store.CreateProviderConfig(ctx, cfg)
}
func (a providerConfigRepoAdapter) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.ProviderConfig, error) {
	return a.Store.GetProviderConfigByID(ctx, tenantID, id)
}
func (a providerConfigRepoAdapter) GetByChannel(ctx context.Context, tenantID uuid.UUID, channel domain.Channel) (*domain.ProviderConfig, error) {
	return a.Store.GetProviderConfigByChannel(ctx, tenantID, channel)
}
func (a providerConfigRepoAdapter) GetByWABAID(ctx context.Context, wabaID string) (*domain.ProviderConfig, error) {
	return a.Store.GetProviderConfigByWABAID(ctx, wabaID)
}
func (a providerConfigRepoAdapter) List(ctx context.Context, tenantID uuid.UUID) ([]domain.ProviderConfig, error) {
	return a.Store.ListProviderConfigs(ctx, tenantID)
}
func (a providerConfigRepoAdapter) Update(ctx context.Context, cfg *domain.ProviderConfig) error {
	return a.Store.UpdateProviderConfig(ctx, cfg)
}
func (a providerConfigRepoAdapter) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return a.Store.DeleteProviderConfig(ctx, tenantID, id)
}

type segmentRepoAdapter struct{ *postgres.Store }

func (a segmentRepoAdapter) Create(ctx context.Context, segment *domain.Segment) error {
	return a.Store.CreateSegment(ctx, segment)
}
func (a segmentRepoAdapter) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Segment, error) {
	return a.Store.GetSegmentByID(ctx, tenantID, id)
}
func (a segmentRepoAdapter) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Segment, error) {
	return a.Store.ListSegments(ctx, tenantID)
}
func (a segmentRepoAdapter) Update(ctx context.Context, segment *domain.Segment) error {
	return a.Store.UpdateSegment(ctx, segment)
}
func (a segmentRepoAdapter) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return a.Store.DeleteSegment(ctx, tenantID, id)
}

type emailTemplateRepoAdapter struct{ *postgres.Store }

func (a emailTemplateRepoAdapter) Create(ctx context.Context, tmpl *domain.EmailTemplate) error {
	return a.Store.CreateEmailTemplate(ctx, tmpl)
}
func (a emailTemplateRepoAdapter) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.EmailTemplate, error) {
	return a.Store.GetEmailTemplateByID(ctx, tenantID, id)
}
func (a emailTemplateRepoAdapter) GetByName(ctx context.Context, tenantID uuid.UUID, name string) (*domain.EmailTemplate, error) {
	return a.Store.GetEmailTemplateByName(ctx, tenantID, name)
}
func (a emailTemplateRepoAdapter) List(ctx context.Context, tenantID uuid.UUID, category *domain.EmailCategory, page, perPage int) ([]domain.EmailTemplate, int, error) {
	return a.Store.ListEmailTemplates(ctx, tenantID, category, page, perPage)
}
func (a emailTemplateRepoAdapter) Update(ctx context.Context, tmpl *domain.EmailTemplate) error {
	return a.Store.UpdateEmailTemplate(ctx, tmpl)
}
func (a emailTemplateRepoAdapter) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return a.Store.DeleteEmailTemplate(ctx, tenantID, id)
}

type emailEventRepoAdapter struct{ *postgres.Store }

func (a emailEventRepoAdapter) Create(ctx context.Context, event *domain.EmailEvent) error {
	return a.Store.CreateEmailEvent(ctx, event)
}
func (a emailEventRepoAdapter) GetByCampaignID(ctx context.Context, tenantID, campaignID uuid.UUID, eventType *domain.EmailEventType, page, perPage int) ([]domain.EmailEvent, int, error) {
	return a.Store.GetEmailEventsByCampaignID(ctx, tenantID, campaignID, eventType, page, perPage)
}
func (a emailEventRepoAdapter) GetStats(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*domain.EmailStats, error) {
	return a.Store.GetStats(ctx, tenantID, from, to)
}
func (a emailEventRepoAdapter) GetCampaignStats(ctx context.Context, tenantID, campaignID uuid.UUID) (*domain.EmailCampaignStats, error) {
	return a.Store.GetEmailCampaignStats(ctx, tenantID, campaignID)
}

type unsubscribeRepoAdapter struct{ *postgres.Store }

func (a unsubscribeRepoAdapter) Create(ctx context.Context, unsub *domain.Unsubscribe) error {
	return a.Store.CreateUnsubscribe(ctx, unsub)
}
func (a unsubscribeRepoAdapter) List(ctx context.Context, tenantID uuid.UUID, scope *domain.UnsubscribeScope, page, perPage int) ([]domain.Unsubscribe, int, error) {
	return a.Store.ListUnsubscribes(ctx, tenantID, scope, page, perPage)
}
func (a unsubscribeRepoAdapter) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return a.Store.DeleteUnsubscribe(ctx, tenantID, id)
}

type suppressionRepoAdapter struct{ *postgres.Store }

func (a suppressionRepoAdapter) Create(ctx context.Context, entry *domain.SuppressionEntry) error {
	return a.Store.CreateSuppression(ctx, entry)
}
func (a suppressionRepoAdapter) List(ctx context.Context, tenantID uuid.UUID, reason *domain.SuppressionReason, page, perPage int) ([]domain.SuppressionEntry, int, error) {
	return a.Store.ListSuppressions(ctx, tenantID, reason, page, perPage)
}
func (a suppressionRepoAdapter) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return a.Store.DeleteSuppression(ctx, tenantID, id)
}

// segmentServiceAdapter maps Store directly to SegmentService
type segmentServiceAdapter struct{ *postgres.Store }

func (a segmentServiceAdapter) Create(ctx context.Context, tenantID uuid.UUID, segment *domain.Segment) error {
	segment.TenantID = tenantID
	return a.Store.CreateSegment(ctx, segment)
}
func (a segmentServiceAdapter) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Segment, error) {
	return a.Store.GetSegmentByID(ctx, tenantID, id)
}
func (a segmentServiceAdapter) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Segment, error) {
	return a.Store.ListSegments(ctx, tenantID)
}
func (a segmentServiceAdapter) Update(ctx context.Context, tenantID, id uuid.UUID, segment *domain.Segment) error {
	segment.ID = id
	segment.TenantID = tenantID
	return a.Store.UpdateSegment(ctx, segment)
}
func (a segmentServiceAdapter) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return a.Store.DeleteSegment(ctx, tenantID, id)
}
func (a segmentServiceAdapter) ResolveContacts(ctx context.Context, tenantID, id uuid.UUID, page, perPage int) ([]domain.Contact, int, error) {
	return a.Store.ResolveContacts(ctx, tenantID, id, page, perPage)
}
