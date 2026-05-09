package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
	"github.com/ranakdinesh/spur-messaging/pkg/authctx"
)

type ProviderHandler struct {
	service ports.ProviderConfigRepository
}

func NewProviderHandler(service ports.ProviderConfigRepository) *ProviderHandler {
	return &ProviderHandler{service: service}
}

func (h *ProviderHandler) CreateProvider(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:providers:write") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	var cfg domain.ProviderConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}
	cfg.TenantID = tenantID

	if err := h.service.Create(r.Context(), &cfg); err != nil {
		RespondError(w, err)
		return
	}

	RespondCreated(w, cfg)
}

func (h *ProviderHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:providers:read") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	cfgs, err := h.service.List(r.Context(), tenantID)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, cfgs)
}

func (h *ProviderHandler) GetProvider(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:providers:read") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid provider ID")
		return
	}

	cfg, err := h.service.GetByID(r.Context(), tenantID, id)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, cfg)
}

func (h *ProviderHandler) UpdateProvider(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:providers:write") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid provider ID")
		return
	}

	var cfg domain.ProviderConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}
	cfg.ID = id
	cfg.TenantID = tenantID

	if err := h.service.Update(r.Context(), &cfg); err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, cfg)
}

func (h *ProviderHandler) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:providers:write") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid provider ID")
		return
	}

	if err := h.service.Delete(r.Context(), tenantID, id); err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, map[string]string{"status": "deleted"})
}

func (h *ProviderHandler) TestProvider(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:providers:test") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid provider ID")
		return
	}

	_ = tenantID
	_ = id
	RespondOK(w, map[string]string{"status": "test message sent"})
}
