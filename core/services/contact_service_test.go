package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

func TestContactServiceOptInRecordsConsentEvidence(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	contactID := uuid.New()
	repo := &contactConsentRepoStub{
		contact: &domain.Contact{
			ID:            contactID,
			TenantID:      tenantID,
			OptInWhatsApp: domain.OptInStatusPending,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
	}
	svc := NewContactService(repo)

	err := svc.OptIn(ctx, tenantID, contactID, domain.ChannelWhatsApp, ports.ConsentEvidence{
		Source:    "website_form",
		Purpose:   "marketing",
		Proof:     "checkbox:v1",
		IPAddress: "203.0.113.10",
		UserAgent: "test-agent",
		Brand:     "citual",
	})
	if err != nil {
		t.Fatalf("OptIn returned error: %v", err)
	}
	if repo.updatedStatus != domain.OptInStatusOptedIn {
		t.Fatalf("expected opted_in update, got %q", repo.updatedStatus)
	}
	if len(repo.records) != 1 {
		t.Fatalf("expected one consent record, got %d", len(repo.records))
	}
	record := repo.records[0]
	if record.Channel != domain.ChannelWhatsApp || record.Status != domain.OptInStatusOptedIn {
		t.Fatalf("unexpected consent record channel/status: %q/%q", record.Channel, record.Status)
	}
	if record.Source != "website_form" || record.Purpose != "marketing" || record.Proof != "checkbox:v1" {
		t.Fatalf("consent evidence was not persisted: %#v", record)
	}
	if record.IPAddress != "203.0.113.10" || record.UserAgent != "test-agent" || record.Brand != "citual" {
		t.Fatalf("request evidence was not persisted: %#v", record)
	}
}

func TestContactServiceOptOutRecordsConsentEvidence(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	contactID := uuid.New()
	repo := &contactConsentRepoStub{
		contact: &domain.Contact{
			ID:         contactID,
			TenantID:   tenantID,
			OptInEmail: domain.OptInStatusOptedIn,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
	}
	svc := NewContactService(repo)

	err := svc.OptOut(ctx, tenantID, contactID, domain.ChannelEmail, ports.ConsentEvidence{Source: "unsubscribe_link"})
	if err != nil {
		t.Fatalf("OptOut returned error: %v", err)
	}
	if repo.updatedStatus != domain.OptInStatusOptedOut {
		t.Fatalf("expected opted_out update, got %q", repo.updatedStatus)
	}
	if len(repo.records) != 1 {
		t.Fatalf("expected one consent record, got %d", len(repo.records))
	}
	if repo.records[0].Status != domain.OptInStatusOptedOut || repo.records[0].Source != "unsubscribe_link" {
		t.Fatalf("unexpected opt-out consent record: %#v", repo.records[0])
	}
}

func TestContactServiceOptInAlreadyOptedInDoesNotDuplicateRecord(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	contactID := uuid.New()
	repo := &contactConsentRepoStub{
		contact: &domain.Contact{
			ID:        contactID,
			TenantID:  tenantID,
			OptInSMS:  domain.OptInStatusOptedIn,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	svc := NewContactService(repo)

	if err := svc.OptIn(ctx, tenantID, contactID, domain.ChannelSMS, ports.ConsentEvidence{}); err != nil {
		t.Fatalf("OptIn returned error: %v", err)
	}
	if len(repo.records) != 0 {
		t.Fatalf("expected no duplicate consent record, got %d", len(repo.records))
	}
}

func TestContactServiceDoubleOptInRequiresConfirmation(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	contactID := uuid.New()
	expiresAt := time.Now().Add(24 * time.Hour)
	repo := &contactConsentRepoStub{
		contact: &domain.Contact{
			ID:            contactID,
			TenantID:      tenantID,
			OptInWhatsApp: domain.OptInStatusPending,
		},
	}
	svc := NewContactService(repo)

	if err := svc.OptIn(ctx, tenantID, contactID, domain.ChannelWhatsApp, ports.ConsentEvidence{
		Source:              "website_form",
		DoubleOptInRequired: true,
		ExpiresAt:           &expiresAt,
	}); err != nil {
		t.Fatalf("OptIn returned error: %v", err)
	}
	if repo.updatedStatus != domain.OptInStatusDoubleOptInPending {
		t.Fatalf("expected double_opt_in_pending update, got %q", repo.updatedStatus)
	}
	if len(repo.records) != 1 || repo.records[0].Status != domain.OptInStatusDoubleOptInPending {
		t.Fatalf("expected pending consent record, got %#v", repo.records)
	}
	if repo.records[0].ExpiresAt == nil {
		t.Fatal("expected consent expiry to be stored")
	}

	if err := svc.ConfirmOptIn(ctx, tenantID, contactID, domain.ChannelWhatsApp, ports.ConsentEvidence{Proof: "otp:123456"}); err != nil {
		t.Fatalf("ConfirmOptIn returned error: %v", err)
	}
	if repo.updatedStatus != domain.OptInStatusOptedIn {
		t.Fatalf("expected opted_in update, got %q", repo.updatedStatus)
	}
	if len(repo.records) != 2 || repo.records[1].Status != domain.OptInStatusOptedIn || repo.records[1].ConfirmedAt == nil {
		t.Fatalf("expected confirmed consent record, got %#v", repo.records)
	}
}

func TestContactServiceInboundKeywordUpdatesConsent(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	contactID := uuid.New()
	phone := "+971501234567"
	repo := &contactConsentRepoStub{
		contact: &domain.Contact{
			ID:            contactID,
			TenantID:      tenantID,
			Phone:         &phone,
			OptInWhatsApp: domain.OptInStatusOptedIn,
		},
	}
	svc := NewContactService(repo)

	action, err := svc.HandleInboundConsentKeyword(ctx, tenantID, domain.ChannelWhatsApp, "971501234567", "إلغاء الاشتراك", ports.ConsentEvidence{Locale: "ar"})
	if err != nil {
		t.Fatalf("HandleInboundConsentKeyword returned error: %v", err)
	}
	if action != domain.ConsentKeywordOptOut {
		t.Fatalf("action = %q, want %q", action, domain.ConsentKeywordOptOut)
	}
	if repo.updatedStatus != domain.OptInStatusOptedOut {
		t.Fatalf("expected opted_out update, got %q", repo.updatedStatus)
	}
	if len(repo.records) != 1 || repo.records[0].Keyword != "إلغاء الاشتراك" || repo.records[0].Locale != "ar" {
		t.Fatalf("keyword evidence was not stored: %#v", repo.records)
	}
}

type contactConsentRepoStub struct {
	contact       *domain.Contact
	updatedStatus domain.OptInStatus
	records       []domain.ConsentRecord
}

func (r *contactConsentRepoStub) Create(context.Context, *domain.Contact) error { return nil }
func (r *contactConsentRepoStub) GetByID(context.Context, uuid.UUID, uuid.UUID) (*domain.Contact, error) {
	return r.contact, nil
}
func (r *contactConsentRepoStub) GetByPhone(_ context.Context, _ uuid.UUID, phone string) (*domain.Contact, error) {
	if r.contact != nil && r.contact.Phone != nil && *r.contact.Phone == phone {
		return r.contact, nil
	}
	return nil, domain.ErrNotFound
}
func (r *contactConsentRepoStub) GetByEmail(context.Context, uuid.UUID, string) (*domain.Contact, error) {
	return nil, domain.ErrNotFound
}
func (r *contactConsentRepoStub) List(context.Context, uuid.UUID, ports.ContactFilter) ([]domain.Contact, int, error) {
	return nil, 0, nil
}
func (r *contactConsentRepoStub) Update(context.Context, *domain.Contact) error { return nil }
func (r *contactConsentRepoStub) Delete(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}
func (r *contactConsentRepoStub) BulkCreate(context.Context, []domain.Contact) (int, error) {
	return 0, nil
}
func (r *contactConsentRepoStub) UpdateOptIn(_ context.Context, _ uuid.UUID, _ uuid.UUID, channel domain.Channel, status domain.OptInStatus) error {
	r.updatedStatus = status
	switch channel {
	case domain.ChannelWhatsApp:
		r.contact.OptInWhatsApp = status
	case domain.ChannelSMS:
		r.contact.OptInSMS = status
	case domain.ChannelEmail:
		r.contact.OptInEmail = status
	}
	return nil
}
func (r *contactConsentRepoStub) CreateConsentRecord(_ context.Context, record *domain.ConsentRecord) error {
	r.records = append(r.records, *record)
	return nil
}
func (r *contactConsentRepoStub) ListConsentRecords(context.Context, uuid.UUID, uuid.UUID, int, int) ([]domain.ConsentRecord, error) {
	return r.records, nil
}
func (r *contactConsentRepoStub) GetBySegment(context.Context, uuid.UUID, uuid.UUID, int, int) ([]domain.Contact, int, error) {
	return nil, 0, nil
}
