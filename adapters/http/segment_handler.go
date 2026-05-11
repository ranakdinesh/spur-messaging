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

type SegmentHandler struct {
	service ports.SegmentService
}

func NewSegmentHandler(service ports.SegmentService) *SegmentHandler {
	return &SegmentHandler{service: service}
}

func (h *SegmentHandler) CreateSegment(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permSegmentsWrite) {
		RespondError(w, domain.ErrForbidden)
		return
	}

	var segment domain.Segment
	if err := json.NewDecoder(r.Body).Decode(&segment); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	if err := validateSegmentRules(segment.Rules); err != nil {
		RespondError(w, err)
		return
	}

	if err := h.service.Create(r.Context(), tenantID, &segment); err != nil {
		RespondError(w, err)
		return
	}

	RespondCreated(w, segment)
}

func (h *SegmentHandler) ListSegments(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permSegmentsRead) {
		RespondError(w, domain.ErrForbidden)
		return
	}

	segments, err := h.service.List(r.Context(), tenantID)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, segments)
}

func (h *SegmentHandler) GetSegment(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permSegmentsRead) {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid ID format")
		return
	}

	segment, err := h.service.GetByID(r.Context(), tenantID, id)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, segment)
}

func (h *SegmentHandler) UpdateSegment(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permSegmentsWrite) {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid ID format")
		return
	}

	var segment domain.Segment
	if err := json.NewDecoder(r.Body).Decode(&segment); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	if err := validateSegmentRules(segment.Rules); err != nil {
		RespondError(w, err)
		return
	}
	segment.ID = id

	if err := h.service.Update(r.Context(), tenantID, id, &segment); err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, segment)
}

func (h *SegmentHandler) DeleteSegment(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permSegmentsWrite) {
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

func (h *SegmentHandler) ResolveContacts(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permSegmentsRead) {
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

	contacts, total, err := h.service.ResolveContacts(r.Context(), tenantID, id, page, perPage)
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
