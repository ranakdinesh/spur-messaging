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

type SuppressionHandler struct {
	service ports.SuppressionService
}

func NewSuppressionHandler(service ports.SuppressionService) *SuppressionHandler {
	return &SuppressionHandler{service: service}
}

func (h *SuppressionHandler) ListSuppressions(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:contacts:read") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}

	var reason *domain.SuppressionReason
	if rStr := q.Get("reason"); rStr != "" {
		re := domain.SuppressionReason(rStr)
		reason = &re
	}

	entries, total, err := h.service.List(r.Context(), tenantID, reason, page, perPage)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondList(w, entries, Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: (total + perPage - 1) / perPage,
	})
}

func (h *SuppressionHandler) AddToSuppression(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:contacts:manage_consent") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	var req struct {
		Email  string                   `json:"email"`
		Reason domain.SuppressionReason `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	if err := h.service.AddToSuppression(r.Context(), tenantID, req.Email, req.Reason); err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, map[string]string{"status": "suppressed"})
}

func (h *SuppressionHandler) RemoveFromSuppression(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:contacts:manage_consent") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid suppression ID")
		return
	}

	if err := h.service.RemoveFromSuppression(r.Context(), tenantID, id); err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, map[string]string{"status": "removed"})
}

func (h *SuppressionHandler) BulkCheckSuppression(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:contacts:read") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	var req struct {
		Emails []string `json:"emails"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	suppressed, err := h.service.BulkCheck(r.Context(), tenantID, req.Emails)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, map[string][]string{"suppressed": suppressed})
}
