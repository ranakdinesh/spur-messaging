package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
)

func TestBillingServiceRecordMessageChargeIsIdempotent(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	messageID := uuid.New()
	repo := &billingRepoStub{
		rate: &domain.RateCard{
			UnitPrice: 0.75,
			Currency:  "INR",
		},
	}
	svc := NewBillingService(repo)

	entry, err := svc.RecordMessageCharge(ctx, domain.UsageCharge{
		TenantID:  tenantID,
		MessageID: messageID,
		Channel:   domain.ChannelWhatsApp,
		Category:  "marketing",
		Country:   "IN",
		Currency:  "INR",
	})
	if err != nil {
		t.Fatalf("RecordMessageCharge() error = %v", err)
	}
	if entry == nil {
		t.Fatal("expected charge entry")
	}
	if entry.Amount != 0.75 || entry.EntryType != domain.WalletLedgerDebit {
		t.Fatalf("unexpected entry: %#v", entry)
	}

	entry, err = svc.RecordMessageCharge(ctx, domain.UsageCharge{
		TenantID:  tenantID,
		MessageID: messageID,
		Channel:   domain.ChannelWhatsApp,
		Currency:  "INR",
	})
	if err != nil {
		t.Fatalf("second RecordMessageCharge() error = %v", err)
	}
	if entry != nil {
		t.Fatalf("expected duplicate charge to be ignored, got %#v", entry)
	}
	if len(repo.entries) != 1 {
		t.Fatalf("ledger entries = %d, want 1", len(repo.entries))
	}
}

func TestBillingServiceBalanceUsesLedgerTypes(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	repo := &billingRepoStub{}
	svc := NewBillingService(repo)

	if _, err := svc.CreditWallet(ctx, tenantID, 100, "inr", "top up", nil); err != nil {
		t.Fatalf("CreditWallet() error = %v", err)
	}
	if _, err := svc.AdjustWallet(ctx, tenantID, -5, "INR", "manual correction", nil); err != nil {
		t.Fatalf("AdjustWallet() error = %v", err)
	}
	cost := 10.5
	if _, err := svc.RecordMessageCharge(ctx, domain.UsageCharge{
		TenantID:     tenantID,
		MessageID:    uuid.New(),
		Channel:      domain.ChannelEmail,
		Currency:     "INR",
		ProviderCost: &cost,
	}); err != nil {
		t.Fatalf("RecordMessageCharge() error = %v", err)
	}

	balance, err := svc.GetWalletBalance(ctx, tenantID, "inr")
	if err != nil {
		t.Fatalf("GetWalletBalance() error = %v", err)
	}
	if balance.CurrentBalance != 84.5 || balance.AvailableBalance != 84.5 {
		t.Fatalf("balance = %#v, want 84.5", balance)
	}
}

type billingRepoStub struct {
	entries []domain.WalletLedgerEntry
	rate    *domain.RateCard
}

func (r *billingRepoStub) CreateWalletLedgerEntry(_ context.Context, entry *domain.WalletLedgerEntry) error {
	r.entries = append(r.entries, *entry)
	return nil
}

func (r *billingRepoStub) ListWalletLedgerEntries(_ context.Context, tenantID uuid.UUID, currency string, _, _ int) ([]domain.WalletLedgerEntry, int, error) {
	var entries []domain.WalletLedgerEntry
	for _, entry := range r.entries {
		if entry.TenantID == tenantID && entry.Currency == currency {
			entries = append(entries, entry)
		}
	}
	return entries, len(entries), nil
}

func (r *billingRepoStub) GetWalletBalance(_ context.Context, tenantID uuid.UUID, currency string) (*domain.WalletBalance, error) {
	var current, reserved float64
	for _, entry := range r.entries {
		if entry.TenantID != tenantID || entry.Currency != currency {
			continue
		}
		switch entry.EntryType {
		case domain.WalletLedgerCredit, domain.WalletLedgerRefund, domain.WalletLedgerAdjustment:
			current += entry.Amount
		case domain.WalletLedgerDebit:
			current -= entry.Amount
		case domain.WalletLedgerHold:
			reserved += entry.Amount
		case domain.WalletLedgerRelease:
			reserved -= entry.Amount
		}
	}
	return &domain.WalletBalance{
		TenantID:         tenantID,
		Currency:         currency,
		CurrentBalance:   current,
		ReservedBalance:  reserved,
		AvailableBalance: current - reserved,
		UpdatedAt:        time.Now(),
	}, nil
}

func (r *billingRepoStub) WalletLedgerReferenceExists(_ context.Context, tenantID uuid.UUID, referenceType string, referenceID uuid.UUID, entryType domain.WalletLedgerEntryType) (bool, error) {
	for _, entry := range r.entries {
		if entry.TenantID == tenantID && entry.ReferenceType == referenceType && entry.ReferenceID != nil && *entry.ReferenceID == referenceID && entry.EntryType == entryType {
			return true, nil
		}
	}
	return false, nil
}

func (r *billingRepoStub) GetActiveRateCard(_ context.Context, _ uuid.UUID, _ domain.Channel, _, _, _ string, _ time.Time) (*domain.RateCard, error) {
	if r.rate == nil {
		return nil, domain.ErrNotFound
	}
	return r.rate, nil
}

func (r *billingRepoStub) CreateRateCard(_ context.Context, rate *domain.RateCard) error {
	r.rate = rate
	return nil
}
