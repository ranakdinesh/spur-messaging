package domain

import (
	"time"

	"github.com/google/uuid"
)

type WalletLedgerEntryType string

const (
	WalletLedgerCredit     WalletLedgerEntryType = "credit"
	WalletLedgerDebit      WalletLedgerEntryType = "debit"
	WalletLedgerHold       WalletLedgerEntryType = "hold"
	WalletLedgerRelease    WalletLedgerEntryType = "release"
	WalletLedgerRefund     WalletLedgerEntryType = "refund"
	WalletLedgerAdjustment WalletLedgerEntryType = "adjustment"
)

type WalletLedgerEntry struct {
	ID            uuid.UUID             `json:"id"`
	TenantID      uuid.UUID             `json:"tenant_id"`
	EntryType     WalletLedgerEntryType `json:"entry_type"`
	Amount        float64               `json:"amount"`
	Currency      string                `json:"currency"`
	Channel       *Channel              `json:"channel,omitempty"`
	Category      string                `json:"category,omitempty"`
	ReferenceType string                `json:"reference_type,omitempty"`
	ReferenceID   *uuid.UUID            `json:"reference_id,omitempty"`
	Description   string                `json:"description,omitempty"`
	Metadata      map[string]string     `json:"metadata,omitempty"`
	CreatedAt     time.Time             `json:"created_at"`
}

type WalletBalance struct {
	TenantID         uuid.UUID `json:"tenant_id"`
	Currency         string    `json:"currency"`
	CurrentBalance   float64   `json:"current_balance"`
	ReservedBalance  float64   `json:"reserved_balance"`
	AvailableBalance float64   `json:"available_balance"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type RateCard struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      *uuid.UUID `json:"tenant_id,omitempty"`
	Channel       Channel    `json:"channel"`
	Category      string     `json:"category"`
	Country       string     `json:"country"`
	Currency      string     `json:"currency"`
	UnitPrice     float64    `json:"unit_price"`
	EffectiveFrom time.Time  `json:"effective_from"`
	EffectiveTo   *time.Time `json:"effective_to,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type UsageCharge struct {
	TenantID     uuid.UUID
	MessageID    uuid.UUID
	CampaignID   *uuid.UUID
	Channel      Channel
	Category     string
	Country      string
	Currency     string
	ProviderCost *float64
	Provider     string
	Description  string
	OccurredAt   time.Time
}
