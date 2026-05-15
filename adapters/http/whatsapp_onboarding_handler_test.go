package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
	"github.com/ranakdinesh/spur-messaging/pkg/authctx"
)

func TestWhatsAppOnboardingRoutesRegistered(t *testing.T) {
	tenantID := uuid.New()
	service := newWhatsAppOnboardingServiceStub()
	router := chi.NewRouter()
	RegisterRoutes(
		router,
		&MessageHandler{},
		&TemplateHandler{},
		&EmailTemplateHandler{},
		&CampaignHandler{},
		&ContactHandler{},
		&ConversationHandler{},
		&TenantWebhookHandler{},
		&BillingHandler{},
		&SegmentHandler{},
		&ProviderHandler{},
		NewWhatsAppOnboardingHandler(service, WhatsAppOnboardingConfig{MetaAppID: "app_123", GraphAPIVersion: "v23.0"}),
		&UnsubscribeHandler{},
		&SuppressionHandler{},
		&AnalyticsHandler{},
	)

	req := httptest.NewRequest(http.MethodGet, "/messaging/whatsapp/onboarding/config", nil)
	req = req.WithContext(authctx.WithAPIKey(req.Context(), tenantID, []string{permProvidersWrite}))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected route to be registered, got status %d body %s", rec.Code, rec.Body.String())
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("secret")) || bytes.Contains(rec.Body.Bytes(), []byte("token")) {
		t.Fatalf("config response exposed sensitive value: %s", rec.Body.String())
	}
}

func TestWhatsAppOnboardingConfigReturnsSafeEmbeddedSignupFields(t *testing.T) {
	tenantID := uuid.New()
	service := newWhatsAppOnboardingServiceStub()
	handler := NewWhatsAppOnboardingHandler(service, WhatsAppOnboardingConfig{
		MetaAppID:            "app_123",
		ConfigID:             "config_456",
		CallbackURL:          "https://api.example.test/messaging/whatsapp/onboarding/callback",
		GraphAPIVersion:      "v23.0",
		RequestedPermissions: []string{"whatsapp_business_management", "whatsapp_business_messaging"},
	})
	req := httptest.NewRequest(http.MethodGet, "/messaging/whatsapp/onboarding/config", nil)
	req = req.WithContext(authctx.WithAPIKey(req.Context(), tenantID, []string{permProvidersWrite}))
	rec := httptest.NewRecorder()

	handler.Config(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected config success, got %d body %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Success bool                             `json:"success"`
		Data    WhatsAppOnboardingConfigResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode config response: %v", err)
	}
	if response.Data.MetaAppID != "app_123" || response.Data.ConfigID != "config_456" {
		t.Fatalf("unexpected embedded signup config: %#v", response.Data)
	}
	if response.Data.CallbackURL == "" || response.Data.RedirectURI == "" || response.Data.GraphAPIVersion != "v23.0" {
		t.Fatalf("missing frontend startup fields: %#v", response.Data)
	}
	if response.Data.State == "" || response.Data.OnboardingSessionID == "" {
		t.Fatalf("expected persisted session and state, got %#v", response.Data)
	}
	sessionID, err := uuid.Parse(response.Data.OnboardingSessionID)
	if err != nil {
		t.Fatalf("onboarding_session_id is not a UUID: %v", err)
	}
	session, ok := service.sessionsByState[response.Data.State]
	if !ok || session.ID != sessionID || session.TenantID != tenantID {
		t.Fatalf("state was not persisted for callback resolution: %#v", service.sessionsByState)
	}
	assertNoSensitiveWhatsAppConfigFields(t, rec.Body.Bytes())
}

func TestWhatsAppOnboardingConfigCreatesUniquePersistedStates(t *testing.T) {
	tenantID := uuid.New()
	service := newWhatsAppOnboardingServiceStub()
	handler := NewWhatsAppOnboardingHandler(service, WhatsAppOnboardingConfig{MetaAppID: "app_123"})

	first := requestWhatsAppOnboardingConfig(t, handler, tenantID)
	second := requestWhatsAppOnboardingConfig(t, handler, tenantID)

	if first.State == "" || second.State == "" || first.State == second.State {
		t.Fatalf("expected unique states, got first=%q second=%q", first.State, second.State)
	}
	if _, ok := service.sessionsByState[first.State]; !ok {
		t.Fatalf("first state was not persisted")
	}
	if _, ok := service.sessionsByState[second.State]; !ok {
		t.Fatalf("second state was not persisted")
	}
}

func TestWhatsAppOnboardingConfigSessionCanBeResolvedByCallback(t *testing.T) {
	tenantID := uuid.New()
	service := newWhatsAppOnboardingServiceStub()
	handler := NewWhatsAppOnboardingHandler(service, WhatsAppOnboardingConfig{MetaAppID: "app_123"})
	config := requestWhatsAppOnboardingConfig(t, handler, tenantID)
	req := httptest.NewRequest(http.MethodPost, "/messaging/whatsapp/onboarding/callback", bytes.NewBufferString(`{"state":"`+config.State+`","code":"oauth-code","waba_id":"waba_123"}`))
	req = req.WithContext(authctx.WithAPIKey(req.Context(), tenantID, []string{permProvidersWrite}))
	rec := httptest.NewRecorder()

	handler.Callback(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected callback to resolve persisted state, got %d body %s", rec.Code, rec.Body.String())
	}
	if !service.callbackCalled {
		t.Fatal("expected callback service to be called")
	}
}

func TestWhatsAppOnboardingRequiresTenantContext(t *testing.T) {
	handler := NewWhatsAppOnboardingHandler(newWhatsAppOnboardingServiceStub(), WhatsAppOnboardingConfig{})
	req := httptest.NewRequest(http.MethodGet, "/messaging/whatsapp/onboarding/config", nil)
	rec := httptest.NewRecorder()

	handler.Config(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized without tenant context, got %d", rec.Code)
	}
}

func TestWhatsAppOnboardingCallbackValidatesState(t *testing.T) {
	tenantID := uuid.New()
	service := newWhatsAppOnboardingServiceStub()
	handler := NewWhatsAppOnboardingHandler(service, WhatsAppOnboardingConfig{})
	req := httptest.NewRequest(http.MethodPost, "/messaging/whatsapp/onboarding/callback", bytes.NewBufferString(`{"code":"oauth-code"}`))
	req = req.WithContext(authctx.WithAPIKey(req.Context(), tenantID, []string{permProvidersWrite}))
	rec := httptest.NewRecorder()

	handler.Callback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected validation error, got %d body %s", rec.Code, rec.Body.String())
	}
	if service.callbackCalled {
		t.Fatal("callback service should not be called for invalid request")
	}
}

func TestWhatsAppOnboardingCallbackDoesNotExposeTokens(t *testing.T) {
	tenantID := uuid.New()
	service := newWhatsAppOnboardingServiceStub()
	_, _ = service.CreateOnboardingSession(context.Background(), tenantID, "state-1")
	service.callbackResult = &ports.WhatsAppOnboardingResult{
		Account: &domain.WhatsAppBusinessAccount{ID: uuid.New(), TenantID: tenantID, WABAID: "waba_123", Name: "Demo"},
		PhoneNumbers: []domain.WhatsAppPhoneNumber{{
			ID:                 uuid.New(),
			TenantID:           tenantID,
			WABAID:             "waba_123",
			PhoneNumberID:      "phone_123",
			DisplayPhoneNumber: "+91 98765 43210",
		}},
	}
	handler := NewWhatsAppOnboardingHandler(service, WhatsAppOnboardingConfig{})
	req := httptest.NewRequest(http.MethodPost, "/messaging/whatsapp/onboarding/callback", bytes.NewBufferString(`{"state":"state-1","code":"secret-oauth-code","waba_id":"waba_123"}`))
	req = req.WithContext(authctx.WithAPIKey(req.Context(), tenantID, []string{permProvidersWrite}))
	rec := httptest.NewRecorder()

	handler.Callback(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected callback success, got %d body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.Bytes()
	if bytes.Contains(body, []byte("secret-oauth-code")) || bytes.Contains(body, []byte("access_token")) {
		t.Fatalf("callback response exposed sensitive data: %s", rec.Body.String())
	}
}

func TestWhatsAppAccountSyncUsesTenantAndAccountID(t *testing.T) {
	tenantID := uuid.New()
	accountID := uuid.New()
	service := newWhatsAppOnboardingServiceStub()
	service.syncAccountResult = &ports.WhatsAppOnboardingResult{
		Account:      &domain.WhatsAppBusinessAccount{ID: accountID, TenantID: tenantID, WABAID: "waba_123"},
		PhoneNumbers: []domain.WhatsAppPhoneNumber{{ID: uuid.New(), TenantID: tenantID, WABAID: "waba_123", PhoneNumberID: "phone_123"}},
	}
	router := chi.NewRouter()
	router.Post("/messaging/whatsapp/accounts/{id}/sync", NewWhatsAppOnboardingHandler(service, WhatsAppOnboardingConfig{}).SyncAccount)
	req := httptest.NewRequest(http.MethodPost, "/messaging/whatsapp/accounts/"+accountID.String()+"/sync", nil)
	req = req.WithContext(authctx.WithAPIKey(req.Context(), tenantID, []string{permProvidersWrite}))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected sync success, got %d body %s", rec.Code, rec.Body.String())
	}
	if service.syncTenantID != tenantID || service.syncAccountID != accountID {
		t.Fatalf("sync called with wrong tenant/account: %s %s", service.syncTenantID, service.syncAccountID)
	}
}

func TestWhatsAppAccountDeleteDoesNotLeakCrossTenant(t *testing.T) {
	tenantID := uuid.New()
	accountID := uuid.New()
	service := newWhatsAppOnboardingServiceStub()
	service.disconnectErr = domain.ErrNotFound
	router := chi.NewRouter()
	router.Delete("/messaging/whatsapp/accounts/{id}", NewWhatsAppOnboardingHandler(service, WhatsAppOnboardingConfig{}).DeleteAccount)
	req := httptest.NewRequest(http.MethodDelete, "/messaging/whatsapp/accounts/"+accountID.String(), nil)
	req = req.WithContext(authctx.WithAPIKey(req.Context(), tenantID, []string{permProvidersWrite}))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected not found for cross-tenant/missing record, got %d body %s", rec.Code, rec.Body.String())
	}
	if service.disconnectTenantID != tenantID || service.disconnectAccountID != accountID {
		t.Fatalf("delete called with wrong tenant/account: %s %s", service.disconnectTenantID, service.disconnectAccountID)
	}
	var response APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Error.Code != "NOT_FOUND" {
		t.Fatalf("expected safe not found response, got %#v", response.Error)
	}
}

type whatsAppOnboardingServiceStub struct {
	sessionsByState map[string]*domain.WhatsAppOnboardingSession
	sessionCounter  int

	callbackCalled bool
	callbackResult *ports.WhatsAppOnboardingResult
	callbackErr    error

	syncTenantID      uuid.UUID
	syncAccountID     uuid.UUID
	syncAccountResult *ports.WhatsAppOnboardingResult
	syncAccountErr    error

	disconnectTenantID  uuid.UUID
	disconnectAccountID uuid.UUID
	disconnectErr       error
}

func newWhatsAppOnboardingServiceStub() *whatsAppOnboardingServiceStub {
	return &whatsAppOnboardingServiceStub{
		sessionsByState: map[string]*domain.WhatsAppOnboardingSession{},
	}
}

func (s *whatsAppOnboardingServiceStub) CreateOnboardingSession(_ context.Context, tenantID uuid.UUID, state string) (*domain.WhatsAppOnboardingSession, error) {
	s.sessionCounter++
	if state == "" {
		state = uuid.NewString()
	}
	session := &domain.WhatsAppOnboardingSession{ID: uuid.New(), TenantID: tenantID, State: state, Status: domain.WhatsAppOnboardingStatusPending}
	s.sessionsByState[state] = session
	return session, nil
}

func (s *whatsAppOnboardingServiceStub) GetOnboardingSession(_ context.Context, tenantID, id uuid.UUID) (*domain.WhatsAppOnboardingSession, error) {
	return &domain.WhatsAppOnboardingSession{ID: id, TenantID: tenantID, Status: domain.WhatsAppOnboardingStatusPending}, nil
}

func (s *whatsAppOnboardingServiceStub) CompleteOnboardingCallback(_ context.Context, tenantID uuid.UUID, req ports.CompleteWhatsAppOnboardingRequest) (*ports.WhatsAppOnboardingResult, error) {
	s.callbackCalled = true
	if _, ok := s.sessionsByState[req.State]; !ok {
		return nil, domain.ErrNotFound
	}
	if s.callbackErr != nil {
		return nil, s.callbackErr
	}
	if s.callbackResult != nil {
		return s.callbackResult, nil
	}
	return &ports.WhatsAppOnboardingResult{Account: &domain.WhatsAppBusinessAccount{ID: uuid.New(), TenantID: tenantID}}, nil
}

func (s *whatsAppOnboardingServiceStub) ExchangeCodeForToken(context.Context, string) (*ports.WhatsAppTokenExchange, error) {
	return &ports.WhatsAppTokenExchange{AccessToken: "token"}, nil
}

func (s *whatsAppOnboardingServiceStub) SyncWABA(_ context.Context, tenantID uuid.UUID, req ports.SyncWhatsAppWABARequest) (*domain.WhatsAppBusinessAccount, error) {
	return &domain.WhatsAppBusinessAccount{ID: uuid.New(), TenantID: tenantID, WABAID: req.WABAID}, nil
}

func (s *whatsAppOnboardingServiceStub) SyncPhoneNumbers(_ context.Context, tenantID uuid.UUID, req ports.SyncWhatsAppPhoneNumbersRequest) ([]domain.WhatsAppPhoneNumber, error) {
	return []domain.WhatsAppPhoneNumber{{ID: uuid.New(), TenantID: tenantID, WABAID: req.WABAID}}, nil
}

func (s *whatsAppOnboardingServiceStub) ListWhatsAppAccounts(_ context.Context, tenantID uuid.UUID, _, _ int) ([]domain.WhatsAppBusinessAccount, int, error) {
	return []domain.WhatsAppBusinessAccount{{ID: uuid.New(), TenantID: tenantID, WABAID: "waba_123"}}, 1, nil
}

func (s *whatsAppOnboardingServiceStub) GetWhatsAppAccount(_ context.Context, tenantID, id uuid.UUID) (*domain.WhatsAppBusinessAccount, error) {
	return &domain.WhatsAppBusinessAccount{ID: id, TenantID: tenantID, WABAID: "waba_123"}, nil
}

func (s *whatsAppOnboardingServiceStub) SyncWhatsAppAccount(_ context.Context, tenantID, id uuid.UUID) (*ports.WhatsAppOnboardingResult, error) {
	s.syncTenantID = tenantID
	s.syncAccountID = id
	if s.syncAccountErr != nil {
		return nil, s.syncAccountErr
	}
	if s.syncAccountResult != nil {
		return s.syncAccountResult, nil
	}
	return &ports.WhatsAppOnboardingResult{Account: &domain.WhatsAppBusinessAccount{ID: id, TenantID: tenantID}}, nil
}

func (s *whatsAppOnboardingServiceStub) DisconnectWhatsAppAccount(_ context.Context, tenantID, id uuid.UUID) error {
	s.disconnectTenantID = tenantID
	s.disconnectAccountID = id
	return s.disconnectErr
}

func (s *whatsAppOnboardingServiceStub) ListWhatsAppPhoneNumbers(_ context.Context, tenantID uuid.UUID, _, _ int) ([]domain.WhatsAppPhoneNumber, int, error) {
	return []domain.WhatsAppPhoneNumber{{ID: uuid.New(), TenantID: tenantID, PhoneNumberID: "phone_123"}}, 1, nil
}

func (s *whatsAppOnboardingServiceStub) SyncWhatsAppPhoneNumber(_ context.Context, tenantID, id uuid.UUID) (*domain.WhatsAppPhoneNumber, error) {
	return &domain.WhatsAppPhoneNumber{ID: id, TenantID: tenantID, PhoneNumberID: "phone_123"}, nil
}

func (s *whatsAppOnboardingServiceStub) DisconnectWABA(context.Context, uuid.UUID, string) error {
	return nil
}

func (s *whatsAppOnboardingServiceStub) GetOnboardingStatus(_ context.Context, tenantID, sessionID uuid.UUID) (*ports.WhatsAppOnboardingStatusResult, error) {
	return &ports.WhatsAppOnboardingStatusResult{
		Session: &domain.WhatsAppOnboardingSession{ID: sessionID, TenantID: tenantID, Status: domain.WhatsAppOnboardingStatusPending, CreatedAt: time.Now()},
	}, nil
}

func requestWhatsAppOnboardingConfig(t *testing.T, handler *WhatsAppOnboardingHandler, tenantID uuid.UUID) WhatsAppOnboardingConfigResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/messaging/whatsapp/onboarding/config", nil)
	req = req.WithContext(authctx.WithAPIKey(req.Context(), tenantID, []string{permProvidersWrite}))
	rec := httptest.NewRecorder()
	handler.Config(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected config success, got %d body %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Success bool                             `json:"success"`
		Data    WhatsAppOnboardingConfigResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode config response: %v", err)
	}
	assertNoSensitiveWhatsAppConfigFields(t, rec.Body.Bytes())
	return response.Data
}

func assertNoSensitiveWhatsAppConfigFields(t *testing.T, body []byte) {
	t.Helper()
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("decode response for sensitive field scan: %v", err)
	}
	encoded := string(body)
	for _, sensitive := range []string{
		"app_secret",
		"access_token",
		"system_user_token",
		"provider_credentials",
		"credentials",
		"secret",
		"token",
	} {
		if bytes.Contains([]byte(encoded), []byte(sensitive)) {
			t.Fatalf("response contains sensitive field/value %q: %s", sensitive, encoded)
		}
	}
}
