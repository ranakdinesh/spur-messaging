package http

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
	"github.com/ranakdinesh/spur-messaging/pkg/authctx"
)

var templateNameRegex = regexp.MustCompile(`^[a-z0-9_]+$`)

func validateTemplateName(name string) error {
	if len(name) < 1 || len(name) > 512 {
		return domain.NewValidationError("name", "template name must be lowercase alphanumeric with underscores")
	}
	if !templateNameRegex.MatchString(name) {
		return domain.NewValidationError("name", "template name must be lowercase alphanumeric with underscores")
	}
	return nil
}

func validateTemplateCategory(cat domain.TemplateCategory) error {
	switch cat {
	case domain.TemplateCategoryMarketing, domain.TemplateCategoryUtility, domain.TemplateCategoryAuthentication:
		return nil
	default:
		return domain.NewValidationError("category", "category must be marketing, utility, or authentication")
	}
}

type TemplateHandler struct {
	service ports.TemplateService
}

func NewTemplateHandler(service ports.TemplateService) *TemplateHandler {
	return &TemplateHandler{service: service}
}

func (h *TemplateHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:templates:write") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	var req ports.CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	if err := validateTemplateName(req.Name); err != nil {
		RespondError(w, err)
		return
	}

	if err := validateTemplateCategory(req.Category); err != nil {
		RespondError(w, err)
		return
	}

	if req.Language == "" {
		RespondError(w, domain.NewValidationError("language", "invalid language code"))
		return
	}

	tmpl, err := h.service.Create(r.Context(), tenantID, req)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondCreated(w, tmpl)
}

func (h *TemplateHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:templates:read") {
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

	var channel *domain.Channel
	if c := q.Get("channel"); c != "" {
		chanVal := domain.Channel(c)
		channel = &chanVal
	}

	var status *domain.TemplateStatus
	if s := q.Get("status"); s != "" {
		statusVal := domain.TemplateStatus(s)
		status = &statusVal
	}

	tmpls, total, err := h.service.List(r.Context(), tenantID, channel, status, page, perPage)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondList(w, tmpls, Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: (total + perPage - 1) / perPage,
	})
}

func (h *TemplateHandler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:templates:read") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid template ID")
		return
	}

	tmpl, err := h.service.GetByID(r.Context(), tenantID, id)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, tmpl)
}

func (h *TemplateHandler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:templates:write") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid template ID")
		return
	}

	var req ports.UpdateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	tmpl, err := h.service.Update(r.Context(), tenantID, id, req)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, tmpl)
}

func (h *TemplateHandler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:templates:write") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid template ID")
		return
	}

	if err := h.service.Delete(r.Context(), tenantID, id); err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, map[string]string{"status": "deleted"})
}

func (h *TemplateHandler) SubmitForApproval(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:templates:submit") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid template ID")
		return
	}

	tmpl, err := h.service.SubmitForApproval(r.Context(), tenantID, id)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, tmpl)
}

func (h *TemplateHandler) SyncStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:templates:read") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid template ID")
		return
	}

	tmpl, err := h.service.SyncStatus(r.Context(), tenantID, id)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, tmpl)
}
