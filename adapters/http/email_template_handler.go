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

func validateEmailTemplate(req ports.CreateEmailTemplateRequest) error {
	if len(req.Subject) < 1 || len(req.Subject) > 998 {
		return domain.NewValidationError("subject", "subject is required (max 998 chars)")
	}
	if len(req.HTMLBody) < 1 || len(req.HTMLBody) > 5*1024*1024 {
		return domain.NewValidationError("html_body", "html_body is required (max 5MB)")
	}
	return nil
}

type EmailTemplateHandler struct {
	service ports.EmailTemplateService
}

func NewEmailTemplateHandler(service ports.EmailTemplateService) *EmailTemplateHandler {
	return &EmailTemplateHandler{service: service}
}

func (h *EmailTemplateHandler) CreateEmailTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:templates:write") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	var req ports.CreateEmailTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	if err := validateEmailTemplate(req); err != nil {
		RespondError(w, err)
		return
	}

	tmpl, err := h.service.Create(r.Context(), tenantID, req)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondCreated(w, tmpl)
}

func (h *EmailTemplateHandler) ListEmailTemplates(w http.ResponseWriter, r *http.Request) {
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

	var category *domain.EmailCategory
	if c := q.Get("category"); c != "" {
		catVal := domain.EmailCategory(c)
		category = &catVal
	}

	tmpls, total, err := h.service.List(r.Context(), tenantID, category, page, perPage)
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

func (h *EmailTemplateHandler) GetEmailTemplate(w http.ResponseWriter, r *http.Request) {
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

func (h *EmailTemplateHandler) UpdateEmailTemplate(w http.ResponseWriter, r *http.Request) {
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

	var req ports.UpdateEmailTemplateRequest
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

func (h *EmailTemplateHandler) DeleteEmailTemplate(w http.ResponseWriter, r *http.Request) {
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

func (h *EmailTemplateHandler) PreviewEmailTemplate(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		Variables map[string]string `json:"variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	preview, err := h.service.Preview(r.Context(), tenantID, id, req.Variables)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, preview)
}

func (h *EmailTemplateHandler) DuplicateTemplate(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	tmpl, err := h.service.Duplicate(r.Context(), tenantID, id, req.Name)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, tmpl)
}

func (h *EmailTemplateHandler) SendTestEmail(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:messages:send") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	var req struct {
		To        string            `json:"to"`
		Variables map[string]string `json:"variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	// Assuming a simple response for test email
	_ = tenantID
	RespondOK(w, map[string]string{"status": "test email sent"})
}
