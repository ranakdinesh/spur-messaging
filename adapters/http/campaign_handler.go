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

type CampaignHandler struct {
	service ports.CampaignService
}

func NewCampaignHandler(service ports.CampaignService) *CampaignHandler {
	return &CampaignHandler{service: service}
}

func (h *CampaignHandler) CreateCampaign(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:campaigns:write") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	var req ports.CreateCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	if err := validateCampaignName(req.Name); err != nil {
		RespondError(w, err)
		return
	}

	if err := validateScheduledAt(req.ScheduledAt); err != nil {
		RespondError(w, err)
		return
	}

	if len(req.ContactIDs) > 100000 {
		RespondError(w, domain.NewValidationError("contact_ids", "campaign limited to 100,000 contacts"))
		return
	}

	campaign, err := h.service.Create(r.Context(), tenantID, req)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondCreated(w, campaign)
}

func (h *CampaignHandler) ListCampaigns(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:campaigns:read") {
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

	var status *domain.CampaignStatus
	if s := q.Get("status"); s != "" {
		st := domain.CampaignStatus(s)
		status = &st
	}

	campaigns, total, err := h.service.List(r.Context(), tenantID, status, page, perPage)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondList(w, campaigns, Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: (total + perPage - 1) / perPage,
	})
}

func (h *CampaignHandler) GetCampaign(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:campaigns:read") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid ID format")
		return
	}

	campaign, err := h.service.GetByID(r.Context(), tenantID, id)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, campaign)
}

func (h *CampaignHandler) UpdateCampaign(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:campaigns:write") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid ID format")
		return
	}

	var req ports.UpdateCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	campaign, err := h.service.Update(r.Context(), tenantID, id, req)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, campaign)
}

func (h *CampaignHandler) DeleteCampaign(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:campaigns:write") {
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

func (h *CampaignHandler) ExecuteCampaign(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:campaigns:execute") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid ID format")
		return
	}

	if err := h.service.Execute(r.Context(), tenantID, id); err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, map[string]string{"status": "started"})
}

func (h *CampaignHandler) PauseCampaign(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:campaigns:execute") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid ID format")
		return
	}

	if err := h.service.Pause(r.Context(), tenantID, id); err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, map[string]string{"status": "paused"})
}

func (h *CampaignHandler) ResumeCampaign(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:campaigns:execute") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid ID format")
		return
	}

	if err := h.service.Resume(r.Context(), tenantID, id); err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, map[string]string{"status": "resumed"})
}

func (h *CampaignHandler) GetCampaignStats(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:campaigns:read") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid ID format")
		return
	}

	stats, err := h.service.GetStats(r.Context(), tenantID, id)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, stats)
}
