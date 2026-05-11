package http

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
	"github.com/ranakdinesh/spur-messaging/pkg/authctx"
)

type ContactHandler struct {
	service ports.ContactService
}

func NewContactHandler(service ports.ContactService) *ContactHandler {
	return &ContactHandler{service: service}
}

func (h *ContactHandler) CreateContact(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permContactsWrite) {
		RespondError(w, domain.ErrForbidden)
		return
	}

	var req ports.CreateContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	if req.Phone != nil && *req.Phone != "" {
		phone, err := validatePhone(*req.Phone)
		if err != nil {
			RespondError(w, err)
			return
		}
		req.Phone = &phone
	}

	if req.Email != nil && *req.Email != "" {
		email, err := validateEmail(*req.Email)
		if err != nil {
			RespondError(w, err)
			return
		}
		req.Email = &email
	}

	if err := validateTags(req.Tags); err != nil {
		RespondError(w, err)
		return
	}

	if err := validateMetadata(req.Attributes); err != nil {
		RespondError(w, err)
		return
	}

	contact, err := h.service.Create(r.Context(), tenantID, req)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondCreated(w, contact)
}

func (h *ContactHandler) ListContacts(w http.ResponseWriter, r *http.Request) {
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

	filter := ports.ContactFilter{
		Page:    page,
		PerPage: perPage,
	}

	if phone := q.Get("phone"); phone != "" {
		filter.Phone = &phone
	}
	if email := q.Get("email"); email != "" {
		filter.Email = &email
	}
	if tag := q.Get("tag"); tag != "" {
		filter.Tag = &tag
	}

	contacts, total, err := h.service.List(r.Context(), tenantID, filter)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondList(w, contacts, Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: (total + perPage - 1) / perPage,
	})
}

func (h *ContactHandler) GetContact(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permContactsRead) {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid ID format")
		return
	}

	contact, err := h.service.GetByID(r.Context(), tenantID, id)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, contact)
}

func (h *ContactHandler) UpdateContact(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permContactsWrite) {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid ID format")
		return
	}

	var req ports.UpdateContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	if req.Phone != nil && *req.Phone != "" {
		phone, err := validatePhone(*req.Phone)
		if err != nil {
			RespondError(w, err)
			return
		}
		req.Phone = &phone
	}

	if req.Email != nil && *req.Email != "" {
		email, err := validateEmail(*req.Email)
		if err != nil {
			RespondError(w, err)
			return
		}
		req.Email = &email
	}

	if req.Tags != nil {
		if err := validateTags(*req.Tags); err != nil {
			RespondError(w, err)
			return
		}
	}

	if req.Attributes != nil {
		if err := validateMetadata(*req.Attributes); err != nil {
			RespondError(w, err)
			return
		}
	}

	contact, err := h.service.Update(r.Context(), tenantID, id, req)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, contact)
}

func (h *ContactHandler) DeleteContact(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permContactsWrite) {
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

func (h *ContactHandler) BulkImport(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permContactsImport) {
		RespondError(w, domain.ErrForbidden)
		return
	}

	var contacts []ports.CreateContactRequest
	if err := json.NewDecoder(r.Body).Decode(&contacts); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	if len(contacts) > 10000 {
		RespondError(w, domain.NewValidationError("contacts", "bulk import limited to 10,000 contacts per request"))
		return
	}

	result, err := h.service.BulkImport(r.Context(), tenantID, contacts)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, result)
}

func (h *ContactHandler) OptIn(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		Channel domain.Channel        `json:"channel"`
		Consent ports.ConsentEvidence `json:"consent"`
		Source  string                `json:"source"`
		Purpose string                `json:"purpose"`
		Proof   string                `json:"proof"`
		Brand   string                `json:"brand"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	consent := normalizeConsentEvidence(req.Consent, req.Source, req.Purpose, req.Proof, req.Brand, r)
	if err := h.service.OptIn(r.Context(), tenantID, id, req.Channel, consent); err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, map[string]string{"status": "opted_in"})
}

func (h *ContactHandler) OptOut(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		Channel domain.Channel        `json:"channel"`
		Consent ports.ConsentEvidence `json:"consent"`
		Source  string                `json:"source"`
		Purpose string                `json:"purpose"`
		Proof   string                `json:"proof"`
		Brand   string                `json:"brand"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	consent := normalizeConsentEvidence(req.Consent, req.Source, req.Purpose, req.Proof, req.Brand, r)
	if err := h.service.OptOut(r.Context(), tenantID, id, req.Channel, consent); err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, map[string]string{"status": "opted_out"})
}

func (h *ContactHandler) ListConsentRecords(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permContactsRead) {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid ID format")
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
	page, perPage, err = validatePagination(page, perPage)
	if err != nil {
		RespondError(w, err)
		return
	}

	records, err := h.service.ListConsentRecords(r.Context(), tenantID, id, page, perPage)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, records)
}

func normalizeConsentEvidence(consent ports.ConsentEvidence, source, purpose, proof, brand string, r *http.Request) ports.ConsentEvidence {
	if consent.Source == "" {
		consent.Source = source
	}
	if consent.Purpose == "" {
		consent.Purpose = purpose
	}
	if consent.Proof == "" {
		consent.Proof = proof
	}
	if consent.Brand == "" {
		consent.Brand = brand
	}
	if consent.IPAddress == "" {
		consent.IPAddress = clientIP(r)
	}
	if consent.UserAgent == "" {
		consent.UserAgent = r.UserAgent()
	}
	return consent
}

func clientIP(r *http.Request) string {
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		if value := r.Header.Get(header); value != "" {
			host, _, err := net.SplitHostPort(value)
			if err == nil {
				return host
			}
			if comma := strings.IndexByte(value, ','); comma >= 0 {
				return strings.TrimSpace(value[:comma])
			}
			return strings.TrimSpace(value)
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
