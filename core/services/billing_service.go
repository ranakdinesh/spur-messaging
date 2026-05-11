package services

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

const (
	defaultBillingCurrency = "INR"
	defaultBillingCountry  = "IN"
	defaultBillingCategory = "service"
)

type BillingService struct {
	repo ports.BillingRepository
	now  func() time.Time
}

func NewBillingService(repo ports.BillingRepository) *BillingService {
	return &BillingService{repo: repo, now: time.Now}
}

func (s *BillingService) GetWalletBalance(ctx context.Context, tenantID uuid.UUID, currency string) (*domain.WalletBalance, error) {
	return s.repo.GetWalletBalance(ctx, tenantID, normalizeCurrency(currency))
}

func (s *BillingService) ListLedger(ctx context.Context, tenantID uuid.UUID, currency string, page, perPage int) ([]domain.WalletLedgerEntry, int, error) {
	page, perPage = normalizePage(page, perPage, defaultWebhookPerPage, maxWebhookPerPage)
	return s.repo.ListWalletLedgerEntries(ctx, tenantID, normalizeCurrency(currency), page, perPage)
}

func (s *BillingService) CreditWallet(ctx context.Context, tenantID uuid.UUID, amount float64, currency, description string, metadata map[string]string) (*domain.WalletLedgerEntry, error) {
	if amount <= 0 {
		return nil, domain.NewValidationError("amount", "amount must be greater than zero")
	}
	return s.createEntry(ctx, domain.WalletLedgerEntry{
		TenantID:    tenantID,
		EntryType:   domain.WalletLedgerCredit,
		Amount:      amount,
		Currency:    normalizeCurrency(currency),
		Description: description,
		Metadata:    metadata,
	})
}

func (s *BillingService) AdjustWallet(ctx context.Context, tenantID uuid.UUID, amount float64, currency, description string, metadata map[string]string) (*domain.WalletLedgerEntry, error) {
	if amount == 0 {
		return nil, domain.NewValidationError("amount", "adjustment amount must not be zero")
	}
	return s.createEntry(ctx, domain.WalletLedgerEntry{
		TenantID:    tenantID,
		EntryType:   domain.WalletLedgerAdjustment,
		Amount:      amount,
		Currency:    normalizeCurrency(currency),
		Description: description,
		Metadata:    metadata,
	})
}

func (s *BillingService) EstimateMessageCost(ctx context.Context, tenantID uuid.UUID, channel domain.Channel, category, country, currency string) (float64, error) {
	category = normalizeBillingCategory(category)
	country = normalizeCountry(country)
	currency = normalizeCurrency(currency)
	rate, err := s.repo.GetActiveRateCard(ctx, tenantID, channel, category, country, currency, s.now())
	if err != nil {
		if err == domain.ErrNotFound {
			return 0, nil
		}
		return 0, err
	}
	return rate.UnitPrice, nil
}

func (s *BillingService) RecordMessageCharge(ctx context.Context, charge domain.UsageCharge) (*domain.WalletLedgerEntry, error) {
	if charge.MessageID == uuid.Nil {
		return nil, domain.NewValidationError("message_id", "message ID is required")
	}
	exists, err := s.repo.WalletLedgerReferenceExists(ctx, charge.TenantID, "message", charge.MessageID, domain.WalletLedgerDebit)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, nil
	}

	currency := normalizeCurrency(charge.Currency)
	amount := 0.0
	if charge.ProviderCost != nil && *charge.ProviderCost > 0 {
		amount = *charge.ProviderCost
	} else {
		estimate, err := s.EstimateMessageCost(ctx, charge.TenantID, charge.Channel, charge.Category, charge.Country, currency)
		if err != nil {
			return nil, err
		}
		amount = estimate
	}
	if amount <= 0 {
		return nil, nil
	}

	category := normalizeBillingCategory(charge.Category)
	country := normalizeCountry(charge.Country)
	description := charge.Description
	if description == "" {
		description = "Message usage charge"
	}
	metadata := map[string]string{
		"country": country,
	}
	if charge.CampaignID != nil {
		metadata["campaign_id"] = charge.CampaignID.String()
	}
	if charge.Provider != "" {
		metadata["provider"] = charge.Provider
	}
	return s.createEntry(ctx, domain.WalletLedgerEntry{
		TenantID:      charge.TenantID,
		EntryType:     domain.WalletLedgerDebit,
		Amount:        amount,
		Currency:      currency,
		Channel:       &charge.Channel,
		Category:      category,
		ReferenceType: "message",
		ReferenceID:   &charge.MessageID,
		Description:   description,
		Metadata:      metadata,
	})
}

func (s *BillingService) CreateRateCard(ctx context.Context, tenantID *uuid.UUID, channel domain.Channel, category, country, currency string, unitPrice float64, effectiveFrom time.Time) (*domain.RateCard, error) {
	if unitPrice < 0 {
		return nil, domain.NewValidationError("unit_price", "unit price must not be negative")
	}
	if channel != domain.ChannelWhatsApp && channel != domain.ChannelSMS && channel != domain.ChannelEmail {
		return nil, domain.NewValidationError("channel", "channel must be whatsapp, sms, or email")
	}
	if effectiveFrom.IsZero() {
		effectiveFrom = s.now().UTC()
	}
	rate := &domain.RateCard{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Channel:       channel,
		Category:      normalizeBillingCategory(category),
		Country:       normalizeCountry(country),
		Currency:      normalizeCurrency(currency),
		UnitPrice:     unitPrice,
		EffectiveFrom: effectiveFrom,
	}
	if err := s.repo.CreateRateCard(ctx, rate); err != nil {
		return nil, err
	}
	return rate, nil
}

func (s *BillingService) createEntry(ctx context.Context, entry domain.WalletLedgerEntry) (*domain.WalletLedgerEntry, error) {
	entry.ID = uuid.New()
	entry.Currency = normalizeCurrency(entry.Currency)
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = s.now().UTC()
	}
	if entry.Metadata == nil {
		entry.Metadata = map[string]string{}
	}
	if err := s.repo.CreateWalletLedgerEntry(ctx, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func normalizeCurrency(currency string) string {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		return defaultBillingCurrency
	}
	return currency
}

func normalizeCountry(country string) string {
	country = strings.ToUpper(strings.TrimSpace(country))
	if country == "" {
		return defaultBillingCountry
	}
	return country
}

func normalizeBillingCategory(category string) string {
	category = strings.ToLower(strings.TrimSpace(category))
	if category == "" {
		return defaultBillingCategory
	}
	return category
}
