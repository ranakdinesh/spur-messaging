package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
	"github.com/ranakdinesh/spur-messaging/pkg/authctx"
)

type BillingHandler struct {
	service ports.BillingService
}

func NewBillingHandler(service ports.BillingService) *BillingHandler {
	return &BillingHandler{service: service}
}

func (h *BillingHandler) WalletBalance(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permBillingRead) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	balance, err := h.service.GetWalletBalance(r.Context(), tenantID, r.URL.Query().Get("currency"))
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondOK(w, balance)
}

func (h *BillingHandler) Ledger(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permBillingRead) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	page, perPage := parsePagination(r)
	entries, total, err := h.service.ListLedger(r.Context(), tenantID, r.URL.Query().Get("currency"), page, perPage)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondList(w, entries, Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages(total, perPage),
	})
}

func (h *BillingHandler) Usage(w http.ResponseWriter, r *http.Request) {
	h.Ledger(w, r)
}

func (h *BillingHandler) CreditWallet(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permBillingManage) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	var req struct {
		Amount      float64           `json:"amount"`
		Currency    string            `json:"currency"`
		Description string            `json:"description"`
		Metadata    map[string]string `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}
	entry, err := h.service.CreditWallet(r.Context(), tenantID, req.Amount, req.Currency, req.Description, req.Metadata)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondCreated(w, entry)
}

func (h *BillingHandler) AdjustWallet(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permBillingManage) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	var req struct {
		Amount      float64           `json:"amount"`
		Currency    string            `json:"currency"`
		Description string            `json:"description"`
		Metadata    map[string]string `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}
	entry, err := h.service.AdjustWallet(r.Context(), tenantID, req.Amount, req.Currency, req.Description, req.Metadata)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondCreated(w, entry)
}

func (h *BillingHandler) CreateRateCard(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permBillingManage) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	var req struct {
		Channel       domain.Channel `json:"channel"`
		Category      string         `json:"category"`
		Country       string         `json:"country"`
		Currency      string         `json:"currency"`
		UnitPrice     float64        `json:"unit_price"`
		EffectiveFrom string         `json:"effective_from"`
		PlatformWide  bool           `json:"platform_wide"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}
	effectiveFrom := time.Time{}
	if req.EffectiveFrom != "" {
		parsed, err := time.Parse(time.RFC3339, req.EffectiveFrom)
		if err != nil {
			RespondValidationError(w, "effective_from", "effective_from must be RFC3339")
			return
		}
		effectiveFrom = parsed
	}
	var rateTenantID *uuid.UUID
	if !req.PlatformWide {
		rateTenantID = &tenantID
	}
	rate, err := h.service.CreateRateCard(r.Context(), rateTenantID, req.Channel, req.Category, req.Country, req.Currency, req.UnitPrice, effectiveFrom)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondCreated(w, rate)
}

func (h *BillingHandler) EstimateMessageCost(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permBillingRead) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	channel := domain.Channel(r.URL.Query().Get("channel"))
	if channel == "" {
		RespondValidationError(w, "channel", "channel is required")
		return
	}
	amount, err := h.service.EstimateMessageCost(r.Context(), tenantID, channel, r.URL.Query().Get("category"), r.URL.Query().Get("country"), r.URL.Query().Get("currency"))
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondOK(w, map[string]any{
		"amount":   amount,
		"channel":  channel,
		"category": r.URL.Query().Get("category"),
		"country":  r.URL.Query().Get("country"),
		"currency": r.URL.Query().Get("currency"),
	})
}
