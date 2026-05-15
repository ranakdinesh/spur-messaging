package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
	"github.com/ranakdinesh/spur-messaging/pkg/authctx"
)

type WhatsAppOnboardingConfig struct {
	MetaAppID            string   `json:"meta_app_id"`
	ConfigID             string   `json:"config_id,omitempty"`
	RedirectURI          string   `json:"redirect_uri,omitempty"`
	CallbackURL          string   `json:"callback_url,omitempty"`
	RequestedPermissions []string `json:"requested_permissions"`
	GraphAPIVersion      string   `json:"graph_api_version"`
	Provider             string   `json:"provider"`
	OAuthProvider        string   `json:"oauth_provider"`
}

type WhatsAppOnboardingConfigResponse struct {
	MetaAppID            string   `json:"meta_app_id"`
	ConfigID             string   `json:"config_id,omitempty"`
	RedirectURI          string   `json:"redirect_uri,omitempty"`
	CallbackURL          string   `json:"callback_url,omitempty"`
	State                string   `json:"state"`
	RequestedPermissions []string `json:"requested_permissions"`
	GraphAPIVersion      string   `json:"graph_api_version"`
	OnboardingSessionID  string   `json:"onboarding_session_id"`
	Provider             string   `json:"provider"`
	OAuthProvider        string   `json:"oauth_provider"`
}

type WhatsAppOnboardingHandler struct {
	service ports.WhatsAppOnboardingService
	config  WhatsAppOnboardingConfig
}

func NewWhatsAppOnboardingHandler(service ports.WhatsAppOnboardingService, config WhatsAppOnboardingConfig) *WhatsAppOnboardingHandler {
	if config.Provider == "" {
		config.Provider = "meta_cloud"
	}
	if config.OAuthProvider == "" {
		config.OAuthProvider = "meta"
	}
	if config.GraphAPIVersion == "" {
		config.GraphAPIVersion = "v23.0"
	}
	if config.CallbackURL == "" {
		config.CallbackURL = config.RedirectURI
	}
	if config.RedirectURI == "" {
		config.RedirectURI = config.CallbackURL
	}
	if len(config.RequestedPermissions) == 0 {
		config.RequestedPermissions = []string{
			"whatsapp_business_management",
			"whatsapp_business_messaging",
			"business_management",
		}
	}
	return &WhatsAppOnboardingHandler{service: service, config: config}
}

func (h *WhatsAppOnboardingHandler) Config(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	if !authctx.HasPermission(r.Context(), permProvidersWrite) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	session, err := h.service.CreateOnboardingSession(r.Context(), tenantID, "")
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondOK(w, WhatsAppOnboardingConfigResponse{
		MetaAppID:            h.config.MetaAppID,
		ConfigID:             h.config.ConfigID,
		RedirectURI:          h.config.RedirectURI,
		CallbackURL:          h.config.CallbackURL,
		State:                session.State,
		RequestedPermissions: append([]string(nil), h.config.RequestedPermissions...),
		GraphAPIVersion:      h.config.GraphAPIVersion,
		OnboardingSessionID:  session.ID.String(),
		Provider:             h.config.Provider,
		OAuthProvider:        h.config.OAuthProvider,
	})
}

func (h *WhatsAppOnboardingHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	if !authctx.HasPermission(r.Context(), permProvidersWrite) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	var req struct {
		State string `json:"state"`
	}
	if err := decodeOptionalJSON(r, &req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}
	session, err := h.service.CreateOnboardingSession(r.Context(), tenantID, req.State)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondCreated(w, session)
}

func (h *WhatsAppOnboardingHandler) Callback(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	if !authctx.HasPermission(r.Context(), permProvidersWrite) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	var req ports.CompleteWhatsAppOnboardingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}
	if strings.TrimSpace(req.State) == "" {
		RespondValidationError(w, "state", "state is required")
		return
	}
	if strings.TrimSpace(req.Code) == "" {
		RespondValidationError(w, "code", "code is required")
		return
	}
	result, err := h.service.CompleteOnboardingCallback(r.Context(), tenantID, req)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondOK(w, result)
}

func (h *WhatsAppOnboardingHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	if !authctx.HasPermission(r.Context(), permProvidersRead) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	id, ok := uuidParam(w, r, "id")
	if !ok {
		return
	}
	status, err := h.service.GetOnboardingStatus(r.Context(), tenantID, id)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondOK(w, status)
}

func (h *WhatsAppOnboardingHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	if !authctx.HasPermission(r.Context(), permProvidersRead) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	page, perPage, err := paginationFromRequest(r)
	if err != nil {
		RespondError(w, err)
		return
	}
	accounts, total, err := h.service.ListWhatsAppAccounts(r.Context(), tenantID, page, perPage)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondList(w, accounts, pagination(page, perPage, total))
}

func (h *WhatsAppOnboardingHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	if !authctx.HasPermission(r.Context(), permProvidersRead) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	id, ok := uuidParam(w, r, "id")
	if !ok {
		return
	}
	account, err := h.service.GetWhatsAppAccount(r.Context(), tenantID, id)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondOK(w, account)
}

func (h *WhatsAppOnboardingHandler) SyncAccount(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	if !authctx.HasPermission(r.Context(), permProvidersWrite) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	id, ok := uuidParam(w, r, "id")
	if !ok {
		return
	}
	result, err := h.service.SyncWhatsAppAccount(r.Context(), tenantID, id)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondOK(w, result)
}

func (h *WhatsAppOnboardingHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	if !authctx.HasPermission(r.Context(), permProvidersWrite) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	id, ok := uuidParam(w, r, "id")
	if !ok {
		return
	}
	if err := h.service.DisconnectWhatsAppAccount(r.Context(), tenantID, id); err != nil {
		RespondError(w, err)
		return
	}
	RespondOK(w, map[string]string{"status": "disconnected"})
}

func (h *WhatsAppOnboardingHandler) ListPhoneNumbers(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	if !authctx.HasPermission(r.Context(), permProvidersRead) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	page, perPage, err := paginationFromRequest(r)
	if err != nil {
		RespondError(w, err)
		return
	}
	phones, total, err := h.service.ListWhatsAppPhoneNumbers(r.Context(), tenantID, page, perPage)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondList(w, phones, pagination(page, perPage, total))
}

func (h *WhatsAppOnboardingHandler) SyncPhoneNumber(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	if !authctx.HasPermission(r.Context(), permProvidersWrite) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	id, ok := uuidParam(w, r, "id")
	if !ok {
		return
	}
	phone, err := h.service.SyncWhatsAppPhoneNumber(r.Context(), tenantID, id)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondOK(w, phone)
}

func tenantFromRequest(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	if !requireTenant(w, r) {
		return uuid.Nil, false
	}
	return authctx.TenantID(r.Context()), true
}

func requireTenant(w http.ResponseWriter, r *http.Request) bool {
	if !authctx.IsAuthenticated(r.Context()) {
		RespondError(w, domain.ErrUnauthorized)
		return false
	}
	return true
}

func uuidParam(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		RespondValidationError(w, name, "invalid ID format")
		return uuid.Nil, false
	}
	return id, true
}

func decodeOptionalJSON(r *http.Request, dest any) error {
	if r.Body == nil || r.ContentLength == 0 {
		return nil
	}
	return json.NewDecoder(r.Body).Decode(dest)
}

func paginationFromRequest(r *http.Request) (int, int, error) {
	page := 1
	perPage := 20
	if raw := r.URL.Query().Get("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, domain.NewValidationError("page", "page must be a number")
		}
		page = parsed
	}
	if raw := r.URL.Query().Get("per_page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, domain.NewValidationError("per_page", "per_page must be a number")
		}
		perPage = parsed
	}
	return validatePagination(page, perPage)
}

func pagination(page, perPage, total int) Pagination {
	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	return Pagination{Page: page, PerPage: perPage, Total: total, TotalPages: totalPages}
}
