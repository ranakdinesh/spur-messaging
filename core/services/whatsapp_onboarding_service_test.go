package services

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

func TestWhatsAppOnboardingServiceCompleteOnboardingCallback(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	now := time.Date(2026, 5, 12, 10, 30, 0, 0, time.UTC)
	repo := newWhatsAppOnboardingRepoStub()
	session := &domain.WhatsAppOnboardingSession{
		ID:        uuid.New(),
		TenantID:  tenantID,
		State:     "callback-state",
		Status:    domain.WhatsAppOnboardingStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	repo.sessions[session.ID] = session
	repo.sessionsByState[session.State] = session
	providerRepo := &whatsAppProviderConfigRepoStub{byChannelErr: domain.ErrNotFound}
	metaClient := &metaOnboardingClientStub{
		token: &ports.WhatsAppTokenExchange{AccessToken: "meta-access-token", WABAID: "waba_123"},
		waba: &ports.WhatsAppBusinessAccountInfo{
			ID:                         "waba_123",
			MetaBusinessID:             "business_456",
			Name:                       "Citual Demo",
			Currency:                   "INR",
			TimezoneID:                 "Asia/Kolkata",
			BusinessVerificationStatus: domain.WhatsAppBusinessVerificationStatusVerified,
		},
		phones: []ports.WhatsAppPhoneNumberInfo{{
			ID:                     "phone_789",
			DisplayPhoneNumber:     "+91 98765 43210",
			VerifiedName:           "Citual",
			QualityRating:          domain.WhatsAppQualityRatingGreen,
			MessagingLimitTier:     "TIER_1K",
			Status:                 domain.WhatsAppPhoneNumberStatusConnected,
			CodeVerificationStatus: domain.WhatsAppCodeVerificationStatusVerified,
		}},
	}
	codec := &credentialCodecStub{ciphertext: []byte("encrypted-whatsapp-credentials")}
	svc := NewWhatsAppOnboardingService(repo, providerRepo, metaClient, codec)
	svc.now = func() time.Time { return now }

	result, err := svc.CompleteOnboardingCallback(ctx, tenantID, ports.CompleteWhatsAppOnboardingRequest{
		State: "callback-state",
		Code:  "oauth-code",
	})
	if err != nil {
		t.Fatalf("CompleteOnboardingCallback returned error: %v", err)
	}
	if result.Session.Status != domain.WhatsAppOnboardingStatusCompleted {
		t.Fatalf("expected completed session, got %q", result.Session.Status)
	}
	if result.Account == nil || result.Account.WABAID != "waba_123" || result.Account.ProviderConfigID == nil {
		t.Fatalf("expected synced WABA account with provider config, got %#v", result.Account)
	}
	if len(result.PhoneNumbers) != 1 || result.PhoneNumbers[0].PhoneNumberID != "phone_789" {
		t.Fatalf("expected synced phone number, got %#v", result.PhoneNumbers)
	}
	if metaClient.exchangeCode != "oauth-code" || metaClient.getWABAID != "waba_123" || metaClient.listPhoneNumbersWABAID != "waba_123" {
		t.Fatalf("Meta client was not called with expected values: %#v", metaClient)
	}
	if providerRepo.created == nil {
		t.Fatal("expected provider config to be created")
	}
	if providerRepo.created.Provider != "meta_cloud" || providerRepo.created.WABAID != "waba_123" || !providerRepo.created.IsActive {
		t.Fatalf("unexpected provider config: %#v", providerRepo.created)
	}
	if providerRepo.updated == nil || providerRepo.updated.PhoneNumberID != "phone_789" || providerRepo.updated.BusinessID != "business_456" || providerRepo.updated.DisplayPhone != "+91 98765 43210" {
		t.Fatalf("expected provider config metadata to be synced, got %#v", providerRepo.updated)
	}
	if bytes.Contains(providerRepo.created.Credentials, []byte("meta-access-token")) {
		t.Fatal("provider config stored raw access token")
	}
	if !bytes.Contains(codec.plaintext, []byte("meta-access-token")) {
		t.Fatalf("expected access token to be passed to encryption, got %s", codec.plaintext)
	}
}

func TestWhatsAppOnboardingServiceSyncWABAUpdatesExisting(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	createdAt := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	repo := newWhatsAppOnboardingRepoStub()
	existing := &domain.WhatsAppBusinessAccount{
		ID:        uuid.New(),
		TenantID:  tenantID,
		WABAID:    "waba_existing",
		Name:      "Old Name",
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
	repo.accountsByWABA[existing.WABAID] = existing
	metaClient := &metaOnboardingClientStub{waba: &ports.WhatsAppBusinessAccountInfo{
		ID:                         "waba_existing",
		MetaBusinessID:             "business_new",
		Name:                       "New Name",
		BusinessVerificationStatus: domain.WhatsAppBusinessVerificationStatusVerified,
	}}
	svc := NewWhatsAppOnboardingService(repo, &whatsAppProviderConfigRepoStub{}, metaClient, &credentialCodecStub{})

	account, err := svc.SyncWABA(ctx, tenantID, ports.SyncWhatsAppWABARequest{
		AccessToken: "token",
		WABAID:      "waba_existing",
	})
	if err != nil {
		t.Fatalf("SyncWABA returned error: %v", err)
	}
	if account.ID != existing.ID {
		t.Fatalf("expected existing account to be updated, got new ID %s", account.ID)
	}
	if account.Name != "New Name" || account.MetaBusinessID != "business_new" {
		t.Fatalf("expected refreshed WABA data, got %#v", account)
	}
	if !account.CreatedAt.Equal(createdAt) {
		t.Fatalf("expected original CreatedAt to be preserved, got %s", account.CreatedAt)
	}
}

func TestWhatsAppOnboardingServiceDisconnectWABA(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	providerConfigID := uuid.New()
	accountID := uuid.New()
	repo := newWhatsAppOnboardingRepoStub()
	repo.accountsByWABA["waba_123"] = &domain.WhatsAppBusinessAccount{
		ID:               accountID,
		TenantID:         tenantID,
		WABAID:           "waba_123",
		ProviderConfigID: &providerConfigID,
	}
	providerRepo := &whatsAppProviderConfigRepoStub{}
	svc := NewWhatsAppOnboardingService(repo, providerRepo, &metaOnboardingClientStub{}, &credentialCodecStub{})

	if err := svc.DisconnectWABA(ctx, tenantID, "waba_123"); err != nil {
		t.Fatalf("DisconnectWABA returned error: %v", err)
	}
	if providerRepo.updatedIsActive == nil || *providerRepo.updatedIsActive {
		t.Fatalf("expected linked provider config to be deactivated, got %#v", providerRepo.updatedIsActive)
	}
	if repo.deletedAccountID != accountID {
		t.Fatalf("expected account %s to be deleted, got %s", accountID, repo.deletedAccountID)
	}
}

func TestWhatsAppOnboardingServiceFailureMarksSessionFailed(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	repo := newWhatsAppOnboardingRepoStub()
	session := &domain.WhatsAppOnboardingSession{
		ID:       uuid.New(),
		TenantID: tenantID,
		State:    "bad-state",
		Status:   domain.WhatsAppOnboardingStatusPending,
	}
	repo.sessions[session.ID] = session
	repo.sessionsByState[session.State] = session
	svc := NewWhatsAppOnboardingService(repo, &whatsAppProviderConfigRepoStub{}, &metaOnboardingClientStub{
		exchangeErr: errors.New("token exchange failed"),
	}, &credentialCodecStub{})

	_, err := svc.CompleteOnboardingCallback(ctx, tenantID, ports.CompleteWhatsAppOnboardingRequest{
		State: "bad-state",
		Code:  "bad-code",
	})
	if err == nil {
		t.Fatal("expected callback completion to fail")
	}
	if session.Status != domain.WhatsAppOnboardingStatusFailed {
		t.Fatalf("expected session to be marked failed, got %q", session.Status)
	}
	if repo.failedMessage != "token exchange failed" {
		t.Fatalf("expected failure message to be stored, got %q", repo.failedMessage)
	}
}

func TestWhatsAppOnboardingServiceNoUsablePhoneDoesNotComplete(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	repo := newWhatsAppOnboardingRepoStub()
	session := &domain.WhatsAppOnboardingSession{ID: uuid.New(), TenantID: tenantID, State: "state-no-phone", Status: domain.WhatsAppOnboardingStatusPending}
	repo.sessions[session.ID] = session
	repo.sessionsByState[session.State] = session
	providerRepo := &whatsAppProviderConfigRepoStub{byChannelErr: domain.ErrNotFound}
	svc := NewWhatsAppOnboardingService(repo, providerRepo, &metaOnboardingClientStub{
		token: &ports.WhatsAppTokenExchange{AccessToken: "token", WABAID: "waba_empty"},
		waba:  &ports.WhatsAppBusinessAccountInfo{ID: "waba_empty", MetaBusinessID: "business_empty", Name: "Empty WABA"},
	}, &credentialCodecStub{})

	_, err := svc.CompleteOnboardingCallback(ctx, tenantID, ports.CompleteWhatsAppOnboardingRequest{State: "state-no-phone", Code: "code"})
	if err == nil {
		t.Fatal("expected onboarding to fail without a usable phone number")
	}
	account := repo.accountsByWABA["waba_empty"]
	if account == nil {
		t.Fatal("expected WABA details to be persisted even when phone sync is empty")
	}
	if account.OnboardingStatus == domain.WhatsAppOnboardingStatusCompleted {
		t.Fatalf("account should not be completed without usable phone, got %q", account.OnboardingStatus)
	}
	if session.Status != domain.WhatsAppOnboardingStatusFailed {
		t.Fatalf("expected session failed, got %q", session.Status)
	}
}

func TestWhatsAppOnboardingServicePartialFailurePersistsWABAAndFailsSession(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	repo := newWhatsAppOnboardingRepoStub()
	session := &domain.WhatsAppOnboardingSession{ID: uuid.New(), TenantID: tenantID, State: "state-partial", Status: domain.WhatsAppOnboardingStatusPending}
	repo.sessions[session.ID] = session
	repo.sessionsByState[session.State] = session
	svc := NewWhatsAppOnboardingService(repo, &whatsAppProviderConfigRepoStub{byChannelErr: domain.ErrNotFound}, &metaOnboardingClientStub{
		token:               &ports.WhatsAppTokenExchange{AccessToken: "token", WABAID: "waba_partial"},
		waba:                &ports.WhatsAppBusinessAccountInfo{ID: "waba_partial", MetaBusinessID: "business_partial", Name: "Partial WABA"},
		listPhoneNumbersErr: domain.ErrForbidden,
	}, &credentialCodecStub{})

	_, err := svc.CompleteOnboardingCallback(ctx, tenantID, ports.CompleteWhatsAppOnboardingRequest{State: "state-partial", Code: "code"})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected phone sync error to be returned safely, got %v", err)
	}
	if repo.accountsByWABA["waba_partial"] == nil {
		t.Fatal("expected WABA details to remain persisted after phone sync failure")
	}
	if session.Status != domain.WhatsAppOnboardingStatusFailed {
		t.Fatalf("expected session failed after partial Meta failure, got %q", session.Status)
	}
}

func TestWhatsAppOnboardingServiceBlocksWABALinkedToAnotherTenant(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	otherTenantID := uuid.New()
	providerRepo := &whatsAppProviderConfigRepoStub{
		byWABA: map[string]*domain.ProviderConfig{
			"waba_taken": {ID: uuid.New(), TenantID: otherTenantID, WABAID: "waba_taken"},
		},
	}
	svc := NewWhatsAppOnboardingService(newWhatsAppOnboardingRepoStub(), providerRepo, &metaOnboardingClientStub{}, &credentialCodecStub{})

	_, err := svc.SyncWABA(ctx, tenantID, ports.SyncWhatsAppWABARequest{AccessToken: "token", WABAID: "waba_taken"})
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("expected conflict for WABA linked to another tenant, got %v", err)
	}
}

func TestWhatsAppOnboardingServiceBlocksPhoneLinkedToAnotherTenant(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	otherTenantID := uuid.New()
	providerRepo := &whatsAppProviderConfigRepoStub{
		byPhoneNumberID: map[string]*domain.ProviderConfig{
			"phone_taken": {ID: uuid.New(), TenantID: otherTenantID, PhoneNumberID: "phone_taken"},
		},
	}
	svc := NewWhatsAppOnboardingService(newWhatsAppOnboardingRepoStub(), providerRepo, &metaOnboardingClientStub{
		phones: []ports.WhatsAppPhoneNumberInfo{{
			ID:                     "phone_taken",
			Status:                 domain.WhatsAppPhoneNumberStatusConnected,
			CodeVerificationStatus: domain.WhatsAppCodeVerificationStatusVerified,
		}},
	}, &credentialCodecStub{})

	_, err := svc.SyncPhoneNumbers(ctx, tenantID, ports.SyncWhatsAppPhoneNumbersRequest{AccessToken: "token", WABAID: "waba_123"})
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("expected conflict for phone linked to another tenant, got %v", err)
	}
}

func TestWhatsAppOnboardingServiceSyncAccountHandlesExpiredTokenSafely(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	providerConfigID := uuid.New()
	repo := newWhatsAppOnboardingRepoStub()
	account := &domain.WhatsAppBusinessAccount{ID: uuid.New(), TenantID: tenantID, WABAID: "waba_123", ProviderConfigID: &providerConfigID}
	repo.accounts[account.ID] = account
	providerRepo := &whatsAppProviderConfigRepoStub{byID: map[uuid.UUID]*domain.ProviderConfig{
		providerConfigID: {ID: providerConfigID, TenantID: tenantID, IsActive: true, Credentials: []byte("not-json")},
	}}
	svc := NewWhatsAppOnboardingService(repo, providerRepo, &metaOnboardingClientStub{}, &credentialCodecStub{})

	_, err := svc.SyncWhatsAppAccount(ctx, tenantID, account.ID)
	if !errors.Is(err, domain.ErrCredentialsExpired) {
		t.Fatalf("expected credentials expired, got %v", err)
	}
}

func newWhatsAppOnboardingRepoStub() *whatsAppOnboardingRepoStub {
	return &whatsAppOnboardingRepoStub{
		sessions:         map[uuid.UUID]*domain.WhatsAppOnboardingSession{},
		sessionsByState:  map[string]*domain.WhatsAppOnboardingSession{},
		accounts:         map[uuid.UUID]*domain.WhatsAppBusinessAccount{},
		accountsByWABA:   map[string]*domain.WhatsAppBusinessAccount{},
		phones:           map[uuid.UUID]*domain.WhatsAppPhoneNumber{},
		phonesByProvider: map[string]*domain.WhatsAppPhoneNumber{},
	}
}

type whatsAppOnboardingRepoStub struct {
	sessions         map[uuid.UUID]*domain.WhatsAppOnboardingSession
	sessionsByState  map[string]*domain.WhatsAppOnboardingSession
	accounts         map[uuid.UUID]*domain.WhatsAppBusinessAccount
	accountsByWABA   map[string]*domain.WhatsAppBusinessAccount
	phones           map[uuid.UUID]*domain.WhatsAppPhoneNumber
	phonesByProvider map[string]*domain.WhatsAppPhoneNumber
	failedMessage    string
	deletedAccountID uuid.UUID
}

func (r *whatsAppOnboardingRepoStub) CreateWhatsAppBusinessAccount(_ context.Context, account *domain.WhatsAppBusinessAccount) error {
	r.accounts[account.ID] = account
	r.accountsByWABA[account.WABAID] = account
	return nil
}

func (r *whatsAppOnboardingRepoStub) GetWhatsAppBusinessAccount(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.WhatsAppBusinessAccount, error) {
	account, ok := r.accounts[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return account, nil
}

func (r *whatsAppOnboardingRepoStub) GetWhatsAppBusinessAccountByWABAID(_ context.Context, _ uuid.UUID, wabaID string) (*domain.WhatsAppBusinessAccount, error) {
	account, ok := r.accountsByWABA[wabaID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return account, nil
}

func (r *whatsAppOnboardingRepoStub) ListWhatsAppBusinessAccounts(_ context.Context, _ uuid.UUID, _, _ int) ([]domain.WhatsAppBusinessAccount, int, error) {
	accounts := make([]domain.WhatsAppBusinessAccount, 0, len(r.accountsByWABA))
	for _, account := range r.accountsByWABA {
		accounts = append(accounts, *account)
	}
	return accounts, len(accounts), nil
}

func (r *whatsAppOnboardingRepoStub) UpdateWhatsAppBusinessAccountStatus(_ context.Context, _ uuid.UUID, id uuid.UUID, businessStatus domain.WhatsAppBusinessVerificationStatus, onboardingStatus domain.WhatsAppOnboardingStatus) (*domain.WhatsAppBusinessAccount, error) {
	account, ok := r.accounts[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	account.BusinessVerificationStatus = businessStatus
	account.OnboardingStatus = onboardingStatus
	return account, nil
}

func (r *whatsAppOnboardingRepoStub) UpdateWhatsAppBusinessAccountSync(_ context.Context, account *domain.WhatsAppBusinessAccount) (*domain.WhatsAppBusinessAccount, error) {
	r.accounts[account.ID] = account
	r.accountsByWABA[account.WABAID] = account
	return account, nil
}

func (r *whatsAppOnboardingRepoStub) DeleteWhatsAppBusinessAccount(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	r.deletedAccountID = id
	delete(r.accounts, id)
	for wabaID, account := range r.accountsByWABA {
		if account.ID == id {
			delete(r.accountsByWABA, wabaID)
		}
	}
	return nil
}

func (r *whatsAppOnboardingRepoStub) CreateWhatsAppPhoneNumber(_ context.Context, phone *domain.WhatsAppPhoneNumber) error {
	r.phones[phone.ID] = phone
	r.phonesByProvider[phone.PhoneNumberID] = phone
	return nil
}

func (r *whatsAppOnboardingRepoStub) GetWhatsAppPhoneNumber(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.WhatsAppPhoneNumber, error) {
	phone, ok := r.phones[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return phone, nil
}

func (r *whatsAppOnboardingRepoStub) GetWhatsAppPhoneNumberByPhoneNumberID(_ context.Context, _ uuid.UUID, phoneNumberID string) (*domain.WhatsAppPhoneNumber, error) {
	phone, ok := r.phonesByProvider[phoneNumberID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return phone, nil
}

func (r *whatsAppOnboardingRepoStub) ListWhatsAppPhoneNumbersByWABA(_ context.Context, _ uuid.UUID, wabaID string, _, _ int) ([]domain.WhatsAppPhoneNumber, int, error) {
	phones := make([]domain.WhatsAppPhoneNumber, 0)
	for _, phone := range r.phonesByProvider {
		if phone.WABAID == wabaID {
			phones = append(phones, *phone)
		}
	}
	return phones, len(phones), nil
}

func (r *whatsAppOnboardingRepoStub) ListWhatsAppPhoneNumbers(_ context.Context, _ uuid.UUID, _, _ int) ([]domain.WhatsAppPhoneNumber, int, error) {
	phones := make([]domain.WhatsAppPhoneNumber, 0, len(r.phonesByProvider))
	for _, phone := range r.phonesByProvider {
		phones = append(phones, *phone)
	}
	return phones, len(phones), nil
}

func (r *whatsAppOnboardingRepoStub) UpdateWhatsAppPhoneNumberStatus(_ context.Context, _ uuid.UUID, id uuid.UUID, quality domain.WhatsAppQualityRating, limitTier string, status domain.WhatsAppPhoneNumberStatus, codeStatus domain.WhatsAppCodeVerificationStatus) (*domain.WhatsAppPhoneNumber, error) {
	phone, ok := r.phones[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	phone.QualityRating = quality
	phone.MessagingLimitTier = limitTier
	phone.Status = status
	phone.CodeVerificationStatus = codeStatus
	return phone, nil
}

func (r *whatsAppOnboardingRepoStub) UpdateWhatsAppPhoneNumberSync(_ context.Context, phone *domain.WhatsAppPhoneNumber) (*domain.WhatsAppPhoneNumber, error) {
	r.phones[phone.ID] = phone
	r.phonesByProvider[phone.PhoneNumberID] = phone
	return phone, nil
}

func (r *whatsAppOnboardingRepoStub) DeleteWhatsAppPhoneNumber(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	delete(r.phones, id)
	for providerID, phone := range r.phonesByProvider {
		if phone.ID == id {
			delete(r.phonesByProvider, providerID)
		}
	}
	return nil
}

func (r *whatsAppOnboardingRepoStub) CreateWhatsAppOnboardingSession(_ context.Context, session *domain.WhatsAppOnboardingSession) error {
	r.sessions[session.ID] = session
	r.sessionsByState[session.State] = session
	return nil
}

func (r *whatsAppOnboardingRepoStub) GetWhatsAppOnboardingSession(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.WhatsAppOnboardingSession, error) {
	session, ok := r.sessions[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return session, nil
}

func (r *whatsAppOnboardingRepoStub) GetWhatsAppOnboardingSessionByState(_ context.Context, _ uuid.UUID, state string) (*domain.WhatsAppOnboardingSession, error) {
	session, ok := r.sessionsByState[state]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return session, nil
}

func (r *whatsAppOnboardingRepoStub) UpdateWhatsAppOnboardingSessionStatus(_ context.Context, _ uuid.UUID, id uuid.UUID, status domain.WhatsAppOnboardingStatus) (*domain.WhatsAppOnboardingSession, error) {
	session, ok := r.sessions[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	session.Status = status
	return session, nil
}

func (r *whatsAppOnboardingRepoStub) CompleteWhatsAppOnboardingSession(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.WhatsAppOnboardingSession, error) {
	session, ok := r.sessions[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	now := time.Now().UTC()
	session.Status = domain.WhatsAppOnboardingStatusCompleted
	session.CompletedAt = &now
	return session, nil
}

func (r *whatsAppOnboardingRepoStub) FailWhatsAppOnboardingSession(_ context.Context, _ uuid.UUID, id uuid.UUID, message string) (*domain.WhatsAppOnboardingSession, error) {
	session, ok := r.sessions[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	r.failedMessage = message
	session.Status = domain.WhatsAppOnboardingStatusFailed
	session.ErrorMessage = &message
	return session, nil
}

type whatsAppProviderConfigRepoStub struct {
	byChannel       *domain.ProviderConfig
	byChannelErr    error
	byID            map[uuid.UUID]*domain.ProviderConfig
	byWABA          map[string]*domain.ProviderConfig
	byPhoneNumberID map[string]*domain.ProviderConfig
	created         *domain.ProviderConfig
	updated         *domain.ProviderConfig

	updatedIsActive *bool
}

func (r *whatsAppProviderConfigRepoStub) Create(_ context.Context, cfg *domain.ProviderConfig) error {
	r.created = cfg
	r.byChannel = cfg
	return nil
}

func (r *whatsAppProviderConfigRepoStub) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.ProviderConfig, error) {
	if r.byID != nil {
		if cfg, ok := r.byID[id]; ok {
			return cfg, nil
		}
	}
	if r.byChannel != nil && r.byChannel.ID == id {
		return r.byChannel, nil
	}
	return nil, domain.ErrNotFound
}

func (r *whatsAppProviderConfigRepoStub) GetByChannel(_ context.Context, _ uuid.UUID, _ domain.Channel) (*domain.ProviderConfig, error) {
	if r.byChannelErr != nil {
		return nil, r.byChannelErr
	}
	if r.byChannel == nil {
		return nil, domain.ErrNotFound
	}
	return r.byChannel, nil
}

func (r *whatsAppProviderConfigRepoStub) GetByWABAID(_ context.Context, wabaID string) (*domain.ProviderConfig, error) {
	if r.byWABA != nil {
		if cfg, ok := r.byWABA[wabaID]; ok {
			return cfg, nil
		}
	}
	if r.byChannel != nil && r.byChannel.WABAID == wabaID {
		return r.byChannel, nil
	}
	return nil, domain.ErrNotFound
}

func (r *whatsAppProviderConfigRepoStub) GetByPhoneNumberID(_ context.Context, phoneNumberID string) (*domain.ProviderConfig, error) {
	if r.byPhoneNumberID != nil {
		if cfg, ok := r.byPhoneNumberID[phoneNumberID]; ok {
			return cfg, nil
		}
	}
	if r.byChannel != nil && r.byChannel.PhoneNumberID == phoneNumberID {
		return r.byChannel, nil
	}
	return nil, domain.ErrNotFound
}

func (r *whatsAppProviderConfigRepoStub) List(_ context.Context, _ uuid.UUID) ([]domain.ProviderConfig, error) {
	if r.byChannel == nil {
		return nil, nil
	}
	return []domain.ProviderConfig{*r.byChannel}, nil
}

func (r *whatsAppProviderConfigRepoStub) Update(_ context.Context, cfg *domain.ProviderConfig) error {
	r.updated = cfg
	r.byChannel = cfg
	if r.byID != nil {
		r.byID[cfg.ID] = cfg
	}
	return nil
}

func (r *whatsAppProviderConfigRepoStub) UpdateIsActive(_ context.Context, _ uuid.UUID, _ uuid.UUID, isActive bool) error {
	r.updatedIsActive = &isActive
	return nil
}

func (r *whatsAppProviderConfigRepoStub) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	r.byChannel = nil
	return nil
}

type metaOnboardingClientStub struct {
	token  *ports.WhatsAppTokenExchange
	waba   *ports.WhatsAppBusinessAccountInfo
	phones []ports.WhatsAppPhoneNumberInfo

	exchangeCode           string
	getWABAID              string
	listPhoneNumbersWABAID string
	exchangeErr            error
	getWABAErr             error
	listPhoneNumbersErr    error
}

func (c *metaOnboardingClientStub) ExchangeCodeForToken(_ context.Context, code string) (*ports.WhatsAppTokenExchange, error) {
	c.exchangeCode = code
	if c.exchangeErr != nil {
		return nil, c.exchangeErr
	}
	return c.token, nil
}

func (c *metaOnboardingClientStub) GetWABA(_ context.Context, _ string, wabaID string) (*ports.WhatsAppBusinessAccountInfo, error) {
	c.getWABAID = wabaID
	if c.getWABAErr != nil {
		return nil, c.getWABAErr
	}
	return c.waba, nil
}

func (c *metaOnboardingClientStub) ListPhoneNumbers(_ context.Context, _ string, wabaID string) ([]ports.WhatsAppPhoneNumberInfo, error) {
	c.listPhoneNumbersWABAID = wabaID
	if c.listPhoneNumbersErr != nil {
		return nil, c.listPhoneNumbersErr
	}
	return c.phones, nil
}

type credentialCodecStub struct {
	ciphertext []byte
	plaintext  []byte
}

func (c *credentialCodecStub) Encrypt(plaintext []byte) ([]byte, error) {
	c.plaintext = append([]byte(nil), plaintext...)
	if c.ciphertext != nil {
		return c.ciphertext, nil
	}
	return []byte("encrypted"), nil
}

func (c *credentialCodecStub) Decrypt(ciphertext []byte) ([]byte, error) {
	return ciphertext, nil
}
