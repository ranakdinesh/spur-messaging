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
	service        ports.ProviderConfigRepository
	messageService ports.MessageService
}

func NewProviderHandler(service ports.ProviderConfigRepository, messageService ports.MessageService) *ProviderHandler {
	return &ProviderHandler{
		service:        service,
		messageService: messageService,
	}
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

	if err := validateChannel(cfg.Channel); err != nil {
		RespondError(w, err)
		return
	}

	// Section 10A.2: One active config per channel per tenant
	existing, err := h.service.GetByChannel(r.Context(), tenantID, cfg.Channel)
	if err == nil && existing != nil && existing.IsActive {
		RespondError(w, domain.NewConflictError("active config for this channel exists"))
		return
	}

	// Section 10A.2: WhatsApp: phone_number_id and waba_id are required
	if cfg.Channel == domain.ChannelWhatsApp {
		if cfg.PhoneNumberID == "" {
			RespondValidationError(w, "phone_number_id", "phone_number_id is required for WhatsApp")
			return
		}
		if cfg.WABAID == "" {
			RespondValidationError(w, "waba_id", "waba_id is required for WhatsApp")
			return
		}
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
		RespondValidationError(w, "id", "invalid ID format")
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
		RespondValidationError(w, "id", "invalid ID format")
		return
	}

	var cfg domain.ProviderConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	if err := validateChannel(cfg.Channel); err != nil {
		RespondError(w, err)
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
		RespondValidationError(w, "id", "invalid ID format")
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
		RespondValidationError(w, "id", "invalid ID format")
		return
	}

	var req struct {
		Recipient string `json:"recipient"`
		Message   string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	pCfg, err := h.service.GetByID(r.Context(), tenantID, id)
	if err != nil {
		RespondError(w, err)
		return
	}

	_, err = h.messageService.Send(r.Context(), tenantID, ports.SendMessageRequest{
		Channel:     pCfg.Channel,
		Recipient:   req.Recipient,
		MessageType: domain.MessageTypeText,
		Text:        &req.Message,
	})
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, map[string]string{"status": "test message sent"})
}
