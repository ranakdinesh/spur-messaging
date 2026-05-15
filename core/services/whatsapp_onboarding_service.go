package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type WhatsAppOnboardingService struct {
	repo         ports.WhatsAppOnboardingRepository
	providerRepo ports.ProviderConfigRepository
	metaClient   ports.WhatsAppMetaOnboardingClient
	credentials  ports.CredentialCodec
	now          func() time.Time
}

func NewWhatsAppOnboardingService(
	repo ports.WhatsAppOnboardingRepository,
	providerRepo ports.ProviderConfigRepository,
	metaClient ports.WhatsAppMetaOnboardingClient,
	credentials ports.CredentialCodec,
) *WhatsAppOnboardingService {
	return &WhatsAppOnboardingService{
		repo:         repo,
		providerRepo: providerRepo,
		metaClient:   metaClient,
		credentials:  credentials,
		now:          time.Now,
	}
}

func (s *WhatsAppOnboardingService) CreateOnboardingSession(ctx context.Context, tenantID uuid.UUID, state string) (*domain.WhatsAppOnboardingSession, error) {
	if tenantID == uuid.Nil {
		return nil, domain.NewValidationError("tenant_id", "tenant ID is required")
	}
	state = strings.TrimSpace(state)
	if state == "" {
		state = uuid.NewString()
	}
	now := s.now().UTC()
	session := &domain.WhatsAppOnboardingSession{
		ID:        uuid.New(),
		TenantID:  tenantID,
		State:     state,
		Status:    domain.WhatsAppOnboardingStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.CreateWhatsAppOnboardingSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *WhatsAppOnboardingService) GetOnboardingSession(ctx context.Context, tenantID, id uuid.UUID) (*domain.WhatsAppOnboardingSession, error) {
	return s.repo.GetWhatsAppOnboardingSession(ctx, tenantID, id)
}

func (s *WhatsAppOnboardingService) CompleteOnboardingCallback(ctx context.Context, tenantID uuid.UUID, req ports.CompleteWhatsAppOnboardingRequest) (*ports.WhatsAppOnboardingResult, error) {
	state := strings.TrimSpace(req.State)
	if state == "" {
		return nil, domain.NewValidationError("state", "state is required")
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return nil, domain.NewValidationError("code", "authorization code is required")
	}

	session, err := s.repo.GetWhatsAppOnboardingSessionByState(ctx, tenantID, state)
	if err != nil {
		return nil, err
	}
	if session.Status == domain.WhatsAppOnboardingStatusCompleted {
		return s.completedResult(ctx, tenantID, session)
	}
	session, err = s.repo.UpdateWhatsAppOnboardingSessionStatus(ctx, tenantID, session.ID, domain.WhatsAppOnboardingStatusInProgress)
	if err != nil {
		return nil, err
	}

	token, err := s.ExchangeCodeForToken(ctx, code)
	if err != nil {
		_, _ = s.repo.FailWhatsAppOnboardingSession(ctx, tenantID, session.ID, err.Error())
		return nil, err
	}
	if token == nil || strings.TrimSpace(token.AccessToken) == "" {
		err := domain.ErrCredentialsExpired
		_, _ = s.repo.FailWhatsAppOnboardingSession(ctx, tenantID, session.ID, err.Error())
		return nil, err
	}
	wabaID := firstNonEmptyString(req.WABAID, token.WABAID)
	if wabaID == "" {
		err := domain.NewValidationError("waba_id", "WABA ID is required")
		_, _ = s.repo.FailWhatsAppOnboardingSession(ctx, tenantID, session.ID, err.Error())
		return nil, err
	}

	providerConfigID, err := s.upsertProviderConfig(ctx, tenantID, wabaID, token.AccessToken)
	if err != nil {
		_, _ = s.repo.FailWhatsAppOnboardingSession(ctx, tenantID, session.ID, err.Error())
		return nil, err
	}
	account, err := s.SyncWABA(ctx, tenantID, ports.SyncWhatsAppWABARequest{
		AccessToken:      token.AccessToken,
		WABAID:           wabaID,
		ProviderConfigID: providerConfigID,
	})
	if err != nil {
		_, _ = s.repo.FailWhatsAppOnboardingSession(ctx, tenantID, session.ID, err.Error())
		return nil, err
	}
	phones, err := s.SyncPhoneNumbers(ctx, tenantID, ports.SyncWhatsAppPhoneNumbersRequest{
		AccessToken: token.AccessToken,
		WABAID:      wabaID,
	})
	if err != nil {
		_, _ = s.repo.FailWhatsAppOnboardingSession(ctx, tenantID, session.ID, err.Error())
		return nil, err
	}
	account, err = s.finalizeWhatsAppAccountSync(ctx, tenantID, account, phones)
	if err != nil {
		_, _ = s.repo.FailWhatsAppOnboardingSession(ctx, tenantID, session.ID, err.Error())
		return nil, err
	}
	session, err = s.repo.CompleteWhatsAppOnboardingSession(ctx, tenantID, session.ID)
	if err != nil {
		return nil, err
	}
	return &ports.WhatsAppOnboardingResult{Session: session, Account: account, PhoneNumbers: phones}, nil
}

func (s *WhatsAppOnboardingService) ExchangeCodeForToken(ctx context.Context, code string) (*ports.WhatsAppTokenExchange, error) {
	if strings.TrimSpace(code) == "" {
		return nil, domain.NewValidationError("code", "authorization code is required")
	}
	return s.metaClient.ExchangeCodeForToken(ctx, code)
}

func (s *WhatsAppOnboardingService) SyncWABA(ctx context.Context, tenantID uuid.UUID, req ports.SyncWhatsAppWABARequest) (*domain.WhatsAppBusinessAccount, error) {
	wabaID := strings.TrimSpace(req.WABAID)
	if wabaID == "" {
		return nil, domain.NewValidationError("waba_id", "WABA ID is required")
	}
	if strings.TrimSpace(req.AccessToken) == "" {
		return nil, domain.ErrCredentialsExpired
	}
	if err := s.ensureWABANotLinkedToAnotherTenant(ctx, tenantID, wabaID); err != nil {
		return nil, err
	}
	info, err := s.metaClient.GetWABA(ctx, req.AccessToken, wabaID)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, domain.NewProviderError("Meta WABA response is empty")
	}
	now := s.now().UTC()
	account := &domain.WhatsAppBusinessAccount{
		ID:                         uuid.New(),
		TenantID:                   tenantID,
		MetaBusinessID:             firstNonEmptyString(info.MetaBusinessID, info.ID),
		WABAID:                     firstNonEmptyString(info.ID, wabaID),
		Name:                       info.Name,
		Currency:                   info.Currency,
		TimezoneID:                 info.TimezoneID,
		BusinessVerificationStatus: defaultBusinessVerificationStatus(info.BusinessVerificationStatus),
		OnboardingStatus:           domain.WhatsAppOnboardingStatusInProgress,
		ProviderConfigID:           req.ProviderConfigID,
		LastSyncedAt:               &now,
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}

	existing, err := s.repo.GetWhatsAppBusinessAccountByWABAID(ctx, tenantID, account.WABAID)
	if err == nil && existing != nil {
		account.ID = existing.ID
		account.CreatedAt = existing.CreatedAt
		return s.repo.UpdateWhatsAppBusinessAccountSync(ctx, account)
	}
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	if err := s.repo.CreateWhatsAppBusinessAccount(ctx, account); err != nil {
		return nil, err
	}
	return account, nil
}

func (s *WhatsAppOnboardingService) SyncPhoneNumbers(ctx context.Context, tenantID uuid.UUID, req ports.SyncWhatsAppPhoneNumbersRequest) ([]domain.WhatsAppPhoneNumber, error) {
	wabaID := strings.TrimSpace(req.WABAID)
	if wabaID == "" {
		return nil, domain.NewValidationError("waba_id", "WABA ID is required")
	}
	if strings.TrimSpace(req.AccessToken) == "" {
		return nil, domain.ErrCredentialsExpired
	}
	infos, err := s.metaClient.ListPhoneNumbers(ctx, req.AccessToken, wabaID)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	phones := make([]domain.WhatsAppPhoneNumber, 0, len(infos))
	for _, info := range infos {
		if strings.TrimSpace(info.ID) == "" {
			continue
		}
		if err := s.ensurePhoneNumberNotLinkedToAnotherTenant(ctx, tenantID, info.ID); err != nil {
			return nil, err
		}
		phone := &domain.WhatsAppPhoneNumber{
			ID:                     uuid.New(),
			TenantID:               tenantID,
			WABAID:                 wabaID,
			PhoneNumberID:          info.ID,
			DisplayPhoneNumber:     info.DisplayPhoneNumber,
			VerifiedName:           info.VerifiedName,
			QualityRating:          defaultQualityRating(info.QualityRating),
			MessagingLimitTier:     info.MessagingLimitTier,
			Status:                 defaultPhoneNumberStatus(info.Status),
			CodeVerificationStatus: defaultCodeVerificationStatus(info.CodeVerificationStatus),
			LastSyncedAt:           &now,
			CreatedAt:              now,
			UpdatedAt:              now,
		}
		existing, err := s.repo.GetWhatsAppPhoneNumberByPhoneNumberID(ctx, tenantID, phone.PhoneNumberID)
		if err == nil && existing != nil {
			phone.ID = existing.ID
			phone.CreatedAt = existing.CreatedAt
			updated, err := s.repo.UpdateWhatsAppPhoneNumberSync(ctx, phone)
			if err != nil {
				return nil, err
			}
			phones = append(phones, *updated)
			continue
		}
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
		if err := s.repo.CreateWhatsAppPhoneNumber(ctx, phone); err != nil {
			return nil, err
		}
		phones = append(phones, *phone)
	}
	return phones, nil
}

func (s *WhatsAppOnboardingService) ListWhatsAppAccounts(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]domain.WhatsAppBusinessAccount, int, error) {
	return s.repo.ListWhatsAppBusinessAccounts(ctx, tenantID, page, perPage)
}

func (s *WhatsAppOnboardingService) GetWhatsAppAccount(ctx context.Context, tenantID, id uuid.UUID) (*domain.WhatsAppBusinessAccount, error) {
	return s.repo.GetWhatsAppBusinessAccount(ctx, tenantID, id)
}

func (s *WhatsAppOnboardingService) SyncWhatsAppAccount(ctx context.Context, tenantID, id uuid.UUID) (*ports.WhatsAppOnboardingResult, error) {
	account, err := s.repo.GetWhatsAppBusinessAccount(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	accessToken, err := s.accessTokenForAccount(ctx, tenantID, account)
	if err != nil {
		return nil, err
	}
	syncedAccount, err := s.SyncWABA(ctx, tenantID, ports.SyncWhatsAppWABARequest{
		AccessToken:      accessToken,
		WABAID:           account.WABAID,
		ProviderConfigID: account.ProviderConfigID,
	})
	if err != nil {
		return nil, err
	}
	phones, err := s.SyncPhoneNumbers(ctx, tenantID, ports.SyncWhatsAppPhoneNumbersRequest{
		AccessToken: accessToken,
		WABAID:      syncedAccount.WABAID,
	})
	if err != nil {
		return nil, err
	}
	syncedAccount, err = s.finalizeWhatsAppAccountSync(ctx, tenantID, syncedAccount, phones)
	if err != nil {
		return nil, err
	}
	return &ports.WhatsAppOnboardingResult{Account: syncedAccount, PhoneNumbers: phones}, nil
}

func (s *WhatsAppOnboardingService) DisconnectWhatsAppAccount(ctx context.Context, tenantID, id uuid.UUID) error {
	account, err := s.repo.GetWhatsAppBusinessAccount(ctx, tenantID, id)
	if err != nil {
		return err
	}
	return s.disconnectAccount(ctx, tenantID, account)
}

func (s *WhatsAppOnboardingService) ListWhatsAppPhoneNumbers(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]domain.WhatsAppPhoneNumber, int, error) {
	return s.repo.ListWhatsAppPhoneNumbers(ctx, tenantID, page, perPage)
}

func (s *WhatsAppOnboardingService) SyncWhatsAppPhoneNumber(ctx context.Context, tenantID, id uuid.UUID) (*domain.WhatsAppPhoneNumber, error) {
	phone, err := s.repo.GetWhatsAppPhoneNumber(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	account, err := s.repo.GetWhatsAppBusinessAccountByWABAID(ctx, tenantID, phone.WABAID)
	if err != nil {
		return nil, err
	}
	accessToken, err := s.accessTokenForAccount(ctx, tenantID, account)
	if err != nil {
		return nil, err
	}
	phones, err := s.SyncPhoneNumbers(ctx, tenantID, ports.SyncWhatsAppPhoneNumbersRequest{
		AccessToken: accessToken,
		WABAID:      account.WABAID,
	})
	if err != nil {
		return nil, err
	}
	for i := range phones {
		if phones[i].ID == phone.ID || phones[i].PhoneNumberID == phone.PhoneNumberID {
			_, _ = s.finalizeWhatsAppAccountSync(ctx, tenantID, account, phones)
			return &phones[i], nil
		}
	}
	return s.repo.GetWhatsAppPhoneNumber(ctx, tenantID, id)
}

func (s *WhatsAppOnboardingService) DisconnectWABA(ctx context.Context, tenantID uuid.UUID, wabaID string) error {
	wabaID = strings.TrimSpace(wabaID)
	if wabaID == "" {
		return domain.NewValidationError("waba_id", "WABA ID is required")
	}
	account, err := s.repo.GetWhatsAppBusinessAccountByWABAID(ctx, tenantID, wabaID)
	if err != nil {
		return err
	}
	return s.disconnectAccount(ctx, tenantID, account)
}

func (s *WhatsAppOnboardingService) disconnectAccount(ctx context.Context, tenantID uuid.UUID, account *domain.WhatsAppBusinessAccount) error {
	if account.ProviderConfigID != nil {
		if err := s.providerRepo.UpdateIsActive(ctx, tenantID, *account.ProviderConfigID, false); err != nil {
			return err
		}
	}
	return s.repo.DeleteWhatsAppBusinessAccount(ctx, tenantID, account.ID)
}

func (s *WhatsAppOnboardingService) GetOnboardingStatus(ctx context.Context, tenantID, sessionID uuid.UUID) (*ports.WhatsAppOnboardingStatusResult, error) {
	session, err := s.repo.GetWhatsAppOnboardingSession(ctx, tenantID, sessionID)
	if err != nil {
		return nil, err
	}
	result := &ports.WhatsAppOnboardingStatusResult{Session: session}
	accounts, _, err := s.repo.ListWhatsAppBusinessAccounts(ctx, tenantID, 1, 1)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return result, nil
	}
	result.Account = &accounts[0]
	phones, _, err := s.repo.ListWhatsAppPhoneNumbersByWABA(ctx, tenantID, accounts[0].WABAID, 1, 100)
	if err != nil {
		return nil, err
	}
	result.PhoneNumbers = phones
	return result, nil
}

func (s *WhatsAppOnboardingService) completedResult(ctx context.Context, tenantID uuid.UUID, session *domain.WhatsAppOnboardingSession) (*ports.WhatsAppOnboardingResult, error) {
	status, err := s.GetOnboardingStatus(ctx, tenantID, session.ID)
	if err != nil {
		return nil, err
	}
	return &ports.WhatsAppOnboardingResult{
		Session:      status.Session,
		Account:      status.Account,
		PhoneNumbers: status.PhoneNumbers,
	}, nil
}

func (s *WhatsAppOnboardingService) upsertProviderConfig(ctx context.Context, tenantID uuid.UUID, wabaID, accessToken string) (*uuid.UUID, error) {
	if err := s.ensureWABANotLinkedToAnotherTenant(ctx, tenantID, wabaID); err != nil {
		return nil, err
	}
	credentials, err := s.encryptWhatsAppCredentials(domain.WhatsAppCredentials{AccessToken: accessToken})
	if err != nil {
		return nil, err
	}
	existing, err := s.providerRepo.GetByChannel(ctx, tenantID, domain.ChannelWhatsApp)
	if err == nil && existing != nil {
		existing.Provider = "meta_cloud"
		existing.Credentials = credentials
		existing.WABAID = wabaID
		existing.IsActive = true
		if err := s.providerRepo.Update(ctx, existing); err != nil {
			return nil, err
		}
		return &existing.ID, nil
	}
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	cfg := &domain.ProviderConfig{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Channel:     domain.ChannelWhatsApp,
		Provider:    "meta_cloud",
		Credentials: credentials,
		IsActive:    true,
		WABAID:      wabaID,
	}
	if err := s.providerRepo.Create(ctx, cfg); err != nil {
		return nil, err
	}
	return &cfg.ID, nil
}

func (s *WhatsAppOnboardingService) finalizeWhatsAppAccountSync(ctx context.Context, tenantID uuid.UUID, account *domain.WhatsAppBusinessAccount, phones []domain.WhatsAppPhoneNumber) (*domain.WhatsAppBusinessAccount, error) {
	usablePhone := firstUsableWhatsAppPhoneNumber(phones)
	if usablePhone == nil {
		if account != nil {
			_, _ = s.repo.UpdateWhatsAppBusinessAccountStatus(ctx, tenantID, account.ID, account.BusinessVerificationStatus, domain.WhatsAppOnboardingStatusInProgress)
		}
		return nil, domain.NewValidationError("phone_numbers", "no usable WhatsApp phone number found for this WABA")
	}
	if err := s.ensurePhoneNumberNotLinkedToAnotherTenant(ctx, tenantID, usablePhone.PhoneNumberID); err != nil {
		return nil, err
	}
	if account.ProviderConfigID != nil {
		cfg, err := s.providerRepo.GetByID(ctx, tenantID, *account.ProviderConfigID)
		if err != nil {
			return nil, err
		}
		cfg.WABAID = account.WABAID
		cfg.BusinessID = account.MetaBusinessID
		cfg.PhoneNumberID = usablePhone.PhoneNumberID
		cfg.DisplayPhone = usablePhone.DisplayPhoneNumber
		cfg.IsActive = true
		if err := s.providerRepo.Update(ctx, cfg); err != nil {
			return nil, err
		}
	}
	return s.repo.UpdateWhatsAppBusinessAccountStatus(ctx, tenantID, account.ID, account.BusinessVerificationStatus, domain.WhatsAppOnboardingStatusCompleted)
}

func (s *WhatsAppOnboardingService) ensureWABANotLinkedToAnotherTenant(ctx context.Context, tenantID uuid.UUID, wabaID string) error {
	if strings.TrimSpace(wabaID) == "" {
		return nil
	}
	cfg, err := s.providerRepo.GetByWABAID(ctx, wabaID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return err
	}
	if cfg.TenantID != tenantID {
		return domain.NewConflictError("WhatsApp Business Account is already linked to another tenant")
	}
	return nil
}

func (s *WhatsAppOnboardingService) ensurePhoneNumberNotLinkedToAnotherTenant(ctx context.Context, tenantID uuid.UUID, phoneNumberID string) error {
	if strings.TrimSpace(phoneNumberID) == "" {
		return nil
	}
	cfg, err := s.providerRepo.GetByPhoneNumberID(ctx, phoneNumberID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return err
	}
	if cfg.TenantID != tenantID {
		return domain.NewConflictError("WhatsApp phone number is already linked to another tenant")
	}
	return nil
}

func (s *WhatsAppOnboardingService) encryptWhatsAppCredentials(credentials domain.WhatsAppCredentials) ([]byte, error) {
	raw, err := json.Marshal(credentials)
	if err != nil {
		return nil, err
	}
	if s.credentials == nil {
		return nil, domain.NewProviderError("credential encryption is not configured")
	}
	return s.credentials.Encrypt(raw)
}

func (s *WhatsAppOnboardingService) accessTokenForAccount(ctx context.Context, tenantID uuid.UUID, account *domain.WhatsAppBusinessAccount) (string, error) {
	if account.ProviderConfigID == nil {
		return "", domain.ErrProviderNotConfigured
	}
	cfg, err := s.providerRepo.GetByID(ctx, tenantID, *account.ProviderConfigID)
	if err != nil {
		return "", err
	}
	if !cfg.IsActive {
		return "", domain.ErrProviderNotConfigured
	}
	if s.credentials == nil {
		return "", domain.NewProviderError("credential encryption is not configured")
	}
	raw, err := s.credentials.Decrypt(cfg.Credentials)
	if err != nil {
		return "", domain.ErrCredentialsExpired
	}
	var credentials domain.WhatsAppCredentials
	if err := json.Unmarshal(raw, &credentials); err != nil {
		return "", domain.ErrCredentialsExpired
	}
	if strings.TrimSpace(credentials.AccessToken) == "" {
		return "", domain.ErrCredentialsExpired
	}
	return credentials.AccessToken, nil
}

func defaultBusinessVerificationStatus(status domain.WhatsAppBusinessVerificationStatus) domain.WhatsAppBusinessVerificationStatus {
	if domain.IsValidWhatsAppBusinessVerificationStatus(status) {
		return status
	}
	return domain.WhatsAppBusinessVerificationStatusUnknown
}

func defaultQualityRating(rating domain.WhatsAppQualityRating) domain.WhatsAppQualityRating {
	if domain.IsValidWhatsAppQualityRating(rating) {
		return rating
	}
	return domain.WhatsAppQualityRatingUnknown
}

func defaultPhoneNumberStatus(status domain.WhatsAppPhoneNumberStatus) domain.WhatsAppPhoneNumberStatus {
	if domain.IsValidWhatsAppPhoneNumberStatus(status) {
		return status
	}
	return domain.WhatsAppPhoneNumberStatusUnknown
}

func defaultCodeVerificationStatus(status domain.WhatsAppCodeVerificationStatus) domain.WhatsAppCodeVerificationStatus {
	if domain.IsValidWhatsAppCodeVerificationStatus(status) {
		return status
	}
	return domain.WhatsAppCodeVerificationStatusUnknown
}

func firstUsableWhatsAppPhoneNumber(phones []domain.WhatsAppPhoneNumber) *domain.WhatsAppPhoneNumber {
	for i := range phones {
		if isUsableWhatsAppPhoneNumber(phones[i]) {
			return &phones[i]
		}
	}
	return nil
}

func isUsableWhatsAppPhoneNumber(phone domain.WhatsAppPhoneNumber) bool {
	if strings.TrimSpace(phone.PhoneNumberID) == "" {
		return false
	}
	switch phone.Status {
	case domain.WhatsAppPhoneNumberStatusConnected:
	default:
		return false
	}
	switch phone.CodeVerificationStatus {
	case domain.WhatsAppCodeVerificationStatusVerified, domain.WhatsAppCodeVerificationStatusUnknown:
		return true
	default:
		return false
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
