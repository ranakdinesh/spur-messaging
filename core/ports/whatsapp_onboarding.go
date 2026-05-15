package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
)

type WhatsAppOnboardingService interface {
	CreateOnboardingSession(ctx context.Context, tenantID uuid.UUID, state string) (*domain.WhatsAppOnboardingSession, error)
	GetOnboardingSession(ctx context.Context, tenantID, id uuid.UUID) (*domain.WhatsAppOnboardingSession, error)
	CompleteOnboardingCallback(ctx context.Context, tenantID uuid.UUID, req CompleteWhatsAppOnboardingRequest) (*WhatsAppOnboardingResult, error)
	ExchangeCodeForToken(ctx context.Context, code string) (*WhatsAppTokenExchange, error)
	SyncWABA(ctx context.Context, tenantID uuid.UUID, req SyncWhatsAppWABARequest) (*domain.WhatsAppBusinessAccount, error)
	SyncPhoneNumbers(ctx context.Context, tenantID uuid.UUID, req SyncWhatsAppPhoneNumbersRequest) ([]domain.WhatsAppPhoneNumber, error)
	ListWhatsAppAccounts(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]domain.WhatsAppBusinessAccount, int, error)
	GetWhatsAppAccount(ctx context.Context, tenantID, id uuid.UUID) (*domain.WhatsAppBusinessAccount, error)
	SyncWhatsAppAccount(ctx context.Context, tenantID, id uuid.UUID) (*WhatsAppOnboardingResult, error)
	DisconnectWhatsAppAccount(ctx context.Context, tenantID, id uuid.UUID) error
	ListWhatsAppPhoneNumbers(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]domain.WhatsAppPhoneNumber, int, error)
	SyncWhatsAppPhoneNumber(ctx context.Context, tenantID, id uuid.UUID) (*domain.WhatsAppPhoneNumber, error)
	DisconnectWABA(ctx context.Context, tenantID uuid.UUID, wabaID string) error
	GetOnboardingStatus(ctx context.Context, tenantID, sessionID uuid.UUID) (*WhatsAppOnboardingStatusResult, error)
}

type WhatsAppOnboardingRepository interface {
	CreateWhatsAppBusinessAccount(ctx context.Context, account *domain.WhatsAppBusinessAccount) error
	GetWhatsAppBusinessAccount(ctx context.Context, tenantID, id uuid.UUID) (*domain.WhatsAppBusinessAccount, error)
	GetWhatsAppBusinessAccountByWABAID(ctx context.Context, tenantID uuid.UUID, wabaID string) (*domain.WhatsAppBusinessAccount, error)
	ListWhatsAppBusinessAccounts(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]domain.WhatsAppBusinessAccount, int, error)
	UpdateWhatsAppBusinessAccountStatus(ctx context.Context, tenantID, id uuid.UUID, businessStatus domain.WhatsAppBusinessVerificationStatus, onboardingStatus domain.WhatsAppOnboardingStatus) (*domain.WhatsAppBusinessAccount, error)
	UpdateWhatsAppBusinessAccountSync(ctx context.Context, account *domain.WhatsAppBusinessAccount) (*domain.WhatsAppBusinessAccount, error)
	DeleteWhatsAppBusinessAccount(ctx context.Context, tenantID, id uuid.UUID) error

	CreateWhatsAppPhoneNumber(ctx context.Context, phone *domain.WhatsAppPhoneNumber) error
	GetWhatsAppPhoneNumber(ctx context.Context, tenantID, id uuid.UUID) (*domain.WhatsAppPhoneNumber, error)
	GetWhatsAppPhoneNumberByPhoneNumberID(ctx context.Context, tenantID uuid.UUID, phoneNumberID string) (*domain.WhatsAppPhoneNumber, error)
	ListWhatsAppPhoneNumbersByWABA(ctx context.Context, tenantID uuid.UUID, wabaID string, page, perPage int) ([]domain.WhatsAppPhoneNumber, int, error)
	ListWhatsAppPhoneNumbers(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]domain.WhatsAppPhoneNumber, int, error)
	UpdateWhatsAppPhoneNumberStatus(ctx context.Context, tenantID, id uuid.UUID, quality domain.WhatsAppQualityRating, limitTier string, status domain.WhatsAppPhoneNumberStatus, codeStatus domain.WhatsAppCodeVerificationStatus) (*domain.WhatsAppPhoneNumber, error)
	UpdateWhatsAppPhoneNumberSync(ctx context.Context, phone *domain.WhatsAppPhoneNumber) (*domain.WhatsAppPhoneNumber, error)
	DeleteWhatsAppPhoneNumber(ctx context.Context, tenantID, id uuid.UUID) error

	CreateWhatsAppOnboardingSession(ctx context.Context, session *domain.WhatsAppOnboardingSession) error
	GetWhatsAppOnboardingSession(ctx context.Context, tenantID, id uuid.UUID) (*domain.WhatsAppOnboardingSession, error)
	GetWhatsAppOnboardingSessionByState(ctx context.Context, tenantID uuid.UUID, state string) (*domain.WhatsAppOnboardingSession, error)
	UpdateWhatsAppOnboardingSessionStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.WhatsAppOnboardingStatus) (*domain.WhatsAppOnboardingSession, error)
	CompleteWhatsAppOnboardingSession(ctx context.Context, tenantID, id uuid.UUID) (*domain.WhatsAppOnboardingSession, error)
	FailWhatsAppOnboardingSession(ctx context.Context, tenantID, id uuid.UUID, message string) (*domain.WhatsAppOnboardingSession, error)
}

type WhatsAppMetaOnboardingClient interface {
	ExchangeCodeForToken(ctx context.Context, code string) (*WhatsAppTokenExchange, error)
	GetWABA(ctx context.Context, accessToken, wabaID string) (*WhatsAppBusinessAccountInfo, error)
	ListPhoneNumbers(ctx context.Context, accessToken, wabaID string) ([]WhatsAppPhoneNumberInfo, error)
}

type CredentialCodec interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

type CompleteWhatsAppOnboardingRequest struct {
	State  string
	Code   string
	WABAID string
}

type SyncWhatsAppWABARequest struct {
	AccessToken      string
	WABAID           string
	ProviderConfigID *uuid.UUID
}

type SyncWhatsAppPhoneNumbersRequest struct {
	AccessToken string
	WABAID      string
}

type WhatsAppOnboardingResult struct {
	Session      *domain.WhatsAppOnboardingSession
	Account      *domain.WhatsAppBusinessAccount
	PhoneNumbers []domain.WhatsAppPhoneNumber
}

type WhatsAppOnboardingStatusResult struct {
	Session      *domain.WhatsAppOnboardingSession
	Account      *domain.WhatsAppBusinessAccount
	PhoneNumbers []domain.WhatsAppPhoneNumber
}

type WhatsAppTokenExchange struct {
	AccessToken string
	TokenType   string
	ExpiresAt   *time.Time
	WABAID      string
}

type WhatsAppBusinessAccountInfo struct {
	ID                         string
	MetaBusinessID             string
	Name                       string
	Currency                   string
	TimezoneID                 string
	BusinessVerificationStatus domain.WhatsAppBusinessVerificationStatus
}

type WhatsAppPhoneNumberInfo struct {
	ID                     string
	DisplayPhoneNumber     string
	VerifiedName           string
	QualityRating          domain.WhatsAppQualityRating
	MessagingLimitTier     string
	Status                 domain.WhatsAppPhoneNumberStatus
	CodeVerificationStatus domain.WhatsAppCodeVerificationStatus
}
