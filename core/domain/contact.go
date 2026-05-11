package domain

import (
	"time"

	"github.com/google/uuid"
)

type OptInStatus string

const (
	OptInStatusPending            OptInStatus = "pending"
	OptInStatusDoubleOptInPending OptInStatus = "double_opt_in_pending"
	OptInStatusOptedIn            OptInStatus = "opted_in"
	OptInStatusOptedOut           OptInStatus = "opted_out"
)

type Contact struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	Phone         *string // E.164 format
	Email         *string
	Name          *string
	Attributes    map[string]string // custom fields
	Tags          []string
	OptInWhatsApp OptInStatus
	OptInSMS      OptInStatus
	OptInEmail    OptInStatus
	OptedInAt     *time.Time
	OptedOutAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ConsentRecord struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	ContactID   uuid.UUID
	Channel     Channel
	Status      OptInStatus
	Source      string
	Purpose     string
	Proof       string
	IPAddress   string
	UserAgent   string
	Brand       string
	Keyword     string
	Locale      string
	ExpiresAt   *time.Time
	ConfirmedAt *time.Time
	CreatedAt   time.Time
}

type ConsentKeywordAction string

const (
	ConsentKeywordUnknown ConsentKeywordAction = ""
	ConsentKeywordOptIn   ConsentKeywordAction = "opt_in"
	ConsentKeywordOptOut  ConsentKeywordAction = "opt_out"
)
