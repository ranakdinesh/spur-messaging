package domain

import (
	"time"

	"github.com/google/uuid"
)

type WhatsAppOnboardingStatus string

const (
	WhatsAppOnboardingStatusPending    WhatsAppOnboardingStatus = "pending"
	WhatsAppOnboardingStatusInProgress WhatsAppOnboardingStatus = "in_progress"
	WhatsAppOnboardingStatusCompleted  WhatsAppOnboardingStatus = "completed"
	WhatsAppOnboardingStatusFailed     WhatsAppOnboardingStatus = "failed"
	WhatsAppOnboardingStatusExpired    WhatsAppOnboardingStatus = "expired"
	WhatsAppOnboardingStatusCancelled  WhatsAppOnboardingStatus = "cancelled"
)

type WhatsAppPhoneNumberStatus string

const (
	WhatsAppPhoneNumberStatusPendingVerification WhatsAppPhoneNumberStatus = "pending_verification"
	WhatsAppPhoneNumberStatusConnected           WhatsAppPhoneNumberStatus = "connected"
	WhatsAppPhoneNumberStatusDisconnected        WhatsAppPhoneNumberStatus = "disconnected"
	WhatsAppPhoneNumberStatusFlagged             WhatsAppPhoneNumberStatus = "flagged"
	WhatsAppPhoneNumberStatusRestricted          WhatsAppPhoneNumberStatus = "restricted"
	WhatsAppPhoneNumberStatusBanned              WhatsAppPhoneNumberStatus = "banned"
	WhatsAppPhoneNumberStatusUnknown             WhatsAppPhoneNumberStatus = "unknown"
)

type WhatsAppBusinessVerificationStatus string

const (
	WhatsAppBusinessVerificationStatusNotVerified WhatsAppBusinessVerificationStatus = "not_verified"
	WhatsAppBusinessVerificationStatusPending     WhatsAppBusinessVerificationStatus = "pending"
	WhatsAppBusinessVerificationStatusVerified    WhatsAppBusinessVerificationStatus = "verified"
	WhatsAppBusinessVerificationStatusRejected    WhatsAppBusinessVerificationStatus = "rejected"
	WhatsAppBusinessVerificationStatusUnknown     WhatsAppBusinessVerificationStatus = "unknown"
)

type WhatsAppQualityRating string

const (
	WhatsAppQualityRatingGreen   WhatsAppQualityRating = "green"
	WhatsAppQualityRatingYellow  WhatsAppQualityRating = "yellow"
	WhatsAppQualityRatingRed     WhatsAppQualityRating = "red"
	WhatsAppQualityRatingUnknown WhatsAppQualityRating = "unknown"
)

type WhatsAppCodeVerificationStatus string

const (
	WhatsAppCodeVerificationStatusNotVerified WhatsAppCodeVerificationStatus = "not_verified"
	WhatsAppCodeVerificationStatusPending     WhatsAppCodeVerificationStatus = "pending"
	WhatsAppCodeVerificationStatusVerified    WhatsAppCodeVerificationStatus = "verified"
	WhatsAppCodeVerificationStatusFailed      WhatsAppCodeVerificationStatus = "failed"
	WhatsAppCodeVerificationStatusExpired     WhatsAppCodeVerificationStatus = "expired"
	WhatsAppCodeVerificationStatusUnknown     WhatsAppCodeVerificationStatus = "unknown"
)

type WhatsAppBusinessAccount struct {
	ID                         uuid.UUID
	TenantID                   uuid.UUID
	MetaBusinessID             string
	WABAID                     string
	Name                       string
	Currency                   string
	TimezoneID                 string
	BusinessVerificationStatus WhatsAppBusinessVerificationStatus
	OnboardingStatus           WhatsAppOnboardingStatus
	ProviderConfigID           *uuid.UUID
	LastSyncedAt               *time.Time
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

type WhatsAppPhoneNumber struct {
	ID                     uuid.UUID
	TenantID               uuid.UUID
	WABAID                 string
	PhoneNumberID          string
	DisplayPhoneNumber     string
	VerifiedName           string
	QualityRating          WhatsAppQualityRating
	MessagingLimitTier     string
	Status                 WhatsAppPhoneNumberStatus
	CodeVerificationStatus WhatsAppCodeVerificationStatus
	LastSyncedAt           *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type WhatsAppOnboardingSession struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	State        string
	Status       WhatsAppOnboardingStatus
	ErrorMessage *string
	CompletedAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func IsValidWhatsAppOnboardingStatus(status WhatsAppOnboardingStatus) bool {
	switch status {
	case WhatsAppOnboardingStatusPending,
		WhatsAppOnboardingStatusInProgress,
		WhatsAppOnboardingStatusCompleted,
		WhatsAppOnboardingStatusFailed,
		WhatsAppOnboardingStatusExpired,
		WhatsAppOnboardingStatusCancelled:
		return true
	default:
		return false
	}
}

func IsValidWhatsAppPhoneNumberStatus(status WhatsAppPhoneNumberStatus) bool {
	switch status {
	case WhatsAppPhoneNumberStatusPendingVerification,
		WhatsAppPhoneNumberStatusConnected,
		WhatsAppPhoneNumberStatusDisconnected,
		WhatsAppPhoneNumberStatusFlagged,
		WhatsAppPhoneNumberStatusRestricted,
		WhatsAppPhoneNumberStatusBanned,
		WhatsAppPhoneNumberStatusUnknown:
		return true
	default:
		return false
	}
}

func IsValidWhatsAppBusinessVerificationStatus(status WhatsAppBusinessVerificationStatus) bool {
	switch status {
	case WhatsAppBusinessVerificationStatusNotVerified,
		WhatsAppBusinessVerificationStatusPending,
		WhatsAppBusinessVerificationStatusVerified,
		WhatsAppBusinessVerificationStatusRejected,
		WhatsAppBusinessVerificationStatusUnknown:
		return true
	default:
		return false
	}
}

func IsValidWhatsAppQualityRating(rating WhatsAppQualityRating) bool {
	switch rating {
	case WhatsAppQualityRatingGreen,
		WhatsAppQualityRatingYellow,
		WhatsAppQualityRatingRed,
		WhatsAppQualityRatingUnknown:
		return true
	default:
		return false
	}
}

func IsValidWhatsAppCodeVerificationStatus(status WhatsAppCodeVerificationStatus) bool {
	switch status {
	case WhatsAppCodeVerificationStatusNotVerified,
		WhatsAppCodeVerificationStatusPending,
		WhatsAppCodeVerificationStatusVerified,
		WhatsAppCodeVerificationStatusFailed,
		WhatsAppCodeVerificationStatusExpired,
		WhatsAppCodeVerificationStatusUnknown:
		return true
	default:
		return false
	}
}
