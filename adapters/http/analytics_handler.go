package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
	"github.com/ranakdinesh/spur-messaging/pkg/authctx"
)

type AnalyticsHandler struct {
	messageService  ports.MessageService
	emailAnalytics  ports.EmailAnalyticsService
	campaignService ports.CampaignService
}

func NewAnalyticsHandler(msgSvc ports.MessageService, emailAn ports.EmailAnalyticsService, campSvc ports.CampaignService) *AnalyticsHandler {
	return &AnalyticsHandler{
		messageService:  msgSvc,
		emailAnalytics:  emailAn,
		campaignService: campSvc,
	}
}

func (h *AnalyticsHandler) MessageAnalytics(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:analytics:read") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	_ = tenantID
	RespondOK(w, map[string]any{"status": "analytics data"})
}

func (h *AnalyticsHandler) DashboardOverview(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:analytics:read") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	_ = tenantID
	RespondOK(w, map[string]any{"overview": "data"})
}

func (h *AnalyticsHandler) EmailOverviewStats(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:analytics:read") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	from, _ := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	to, _ := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	if to.IsZero() {
		to = time.Now()
	}

	stats, err := h.emailAnalytics.GetOverview(r.Context(), tenantID, from, to)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, stats)
}

func (h *AnalyticsHandler) EmailCampaignReport(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:analytics:read") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid ID format")
		return
	}

	report, err := h.emailAnalytics.GetCampaignReport(r.Context(), tenantID, id)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, report)
}

func (h *AnalyticsHandler) BounceReport(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:analytics:read") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	from, _ := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	to, _ := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	if to.IsZero() {
		to = time.Now()
	}

	report, err := h.emailAnalytics.GetBounceReport(r.Context(), tenantID, from, to)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, report)
}

func (h *AnalyticsHandler) DomainReputation(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:analytics:read") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	reputation, err := h.emailAnalytics.GetDomainReputation(r.Context(), tenantID)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, reputation)
}

func (h *AnalyticsHandler) TopLinksReport(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:analytics:read") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid ID format")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 10
	}

	links, err := h.emailAnalytics.GetTopLinks(r.Context(), tenantID, id, limit)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, links)
}

func (h *AnalyticsHandler) EngagementByHour(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:analytics:read") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	from, _ := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	to, _ := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	if to.IsZero() {
		to = time.Now()
	}

	engagement, err := h.emailAnalytics.GetEngagementByHour(r.Context(), tenantID, from, to)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, engagement)
}
