package http

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
	"github.com/ranakdinesh/spur-messaging/pkg/authctx"
)

type UnsubscribeHandler struct {
	service ports.UnsubscribeService
}

func NewUnsubscribeHandler(service ports.UnsubscribeService) *UnsubscribeHandler {
	return &UnsubscribeHandler{service: service}
}

func (h *UnsubscribeHandler) ListUnsubscribes(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permContactsRead) {
		RespondError(w, domain.ErrForbidden)
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	if page == 0 {
		page = 1
	}
	if perPage == 0 {
		perPage = 20
	}

	page, perPage, err := validatePagination(page, perPage)
	if err != nil {
		RespondError(w, err)
		return
	}

	var scope *domain.UnsubscribeScope
	if s := q.Get("scope"); s != "" {
		sc := domain.UnsubscribeScope(s)
		scope = &sc
	}

	unsubs, total, err := h.service.List(r.Context(), tenantID, scope, page, perPage)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondList(w, unsubs, Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: (total + perPage - 1) / perPage,
	})
}

func (h *UnsubscribeHandler) Resubscribe(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permContactsManageConsent) {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid ID format")
		return
	}

	if err := h.service.Resubscribe(r.Context(), tenantID, id); err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, map[string]string{"status": "resubscribed"})
}

func (h *UnsubscribeHandler) CheckUnsubscribe(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permContactsRead) {
		RespondError(w, domain.ErrForbidden)
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		RespondValidationError(w, "email", "email is required")
		return
	}

	unsubscribed, err := h.service.IsUnsubscribed(r.Context(), tenantID, email)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, map[string]bool{"unsubscribed": unsubscribed})
}

func (h *UnsubscribeHandler) HandleUnsubscribeLink(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		RespondValidationError(w, "token", "token is required")
		return
	}

	if err := h.service.HandleUnsubscribeLink(r.Context(), token); err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, map[string]string{"status": "unsubscribed"})
}
