package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
	"github.com/ranakdinesh/spur-messaging/pkg/authctx"
)

type TenantWebhookHandler struct {
	service ports.TenantWebhookService
}

func NewTenantWebhookHandler(service ports.TenantWebhookService) *TenantWebhookHandler {
	return &TenantWebhookHandler{service: service}
}

func (h *TenantWebhookHandler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permWebhooksWrite) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	var req ports.CreateWebhookEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}
	endpoint, err := h.service.CreateEndpoint(r.Context(), tenantID, req)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondCreated(w, endpoint)
}

func (h *TenantWebhookHandler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permWebhooksRead) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	page, perPage := parsePagination(r)
	endpoints, total, err := h.service.ListEndpoints(r.Context(), tenantID, page, perPage)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondList(w, endpoints, Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages(total, perPage),
	})
}

func (h *TenantWebhookHandler) GetWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permWebhooksRead) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid ID format")
		return
	}
	endpoint, err := h.service.GetEndpoint(r.Context(), tenantID, id)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondOK(w, endpoint)
}

func (h *TenantWebhookHandler) UpdateWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permWebhooksWrite) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid ID format")
		return
	}
	var req ports.UpdateWebhookEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}
	endpoint, err := h.service.UpdateEndpoint(r.Context(), tenantID, id, req)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondOK(w, endpoint)
}

func (h *TenantWebhookHandler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permWebhooksWrite) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid ID format")
		return
	}
	if err := h.service.DeleteEndpoint(r.Context(), tenantID, id); err != nil {
		RespondError(w, err)
		return
	}
	RespondOK(w, map[string]string{"status": "deleted"})
}

func (h *TenantWebhookHandler) TestWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permWebhooksTest) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid ID format")
		return
	}
	delivery, err := h.service.TestEndpoint(r.Context(), tenantID, id)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondOK(w, delivery)
}

func (h *TenantWebhookHandler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permWebhooksRead) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	var webhookID *uuid.UUID
	if raw := chi.URLParam(r, "id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			RespondValidationError(w, "id", "invalid ID format")
			return
		}
		webhookID = &id
	}
	page, perPage := parsePagination(r)
	deliveries, total, err := h.service.ListDeliveries(r.Context(), tenantID, webhookID, page, perPage)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondList(w, deliveries, Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages(total, perPage),
	})
}

func (h *TenantWebhookHandler) ReplayDelivery(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permWebhooksReplay) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "deliveryID"))
	if err != nil {
		RespondValidationError(w, "delivery_id", "invalid ID format")
		return
	}
	delivery, err := h.service.ReplayDelivery(r.Context(), tenantID, id)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondOK(w, delivery)
}

func parsePagination(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if page < 0 {
		page = 0
	}
	if perPage <= 0 {
		perPage = 25
	}
	return page, perPage
}

func totalPages(total, perPage int) int {
	if perPage <= 0 || total == 0 {
		return 0
	}
	return (total + perPage - 1) / perPage
}
