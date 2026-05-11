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

type ConversationHandler struct {
	service ports.ConversationService
}

func NewConversationHandler(service ports.ConversationService) *ConversationHandler {
	return &ConversationHandler{service: service}
}

func (h *ConversationHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permConversationsRead) {
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

	filter := ports.ConversationFilter{Page: page, PerPage: perPage}
	if channel := q.Get("channel"); channel != "" {
		c := domain.Channel(channel)
		if err := validateChannel(c); err != nil {
			RespondError(w, err)
			return
		}
		filter.Channel = &c
	}
	if status := q.Get("status"); status != "" {
		s := domain.ConversationStatus(status)
		if !domain.IsValidConversationStatus(s) {
			RespondValidationError(w, "status", "invalid conversation status")
			return
		}
		filter.Status = &s
	}
	if handoff := q.Get("handoff_status"); handoff != "" {
		h := domain.ConversationHandoffStatus(handoff)
		if !domain.IsValidConversationHandoffStatus(h) {
			RespondValidationError(w, "handoff_status", "invalid handoff status")
			return
		}
		filter.HandoffStatus = &h
	}
	if assigned := q.Get("assigned_agent_id"); assigned != "" {
		id, err := uuid.Parse(assigned)
		if err != nil {
			RespondValidationError(w, "assigned_agent_id", "invalid ID format")
			return
		}
		filter.AssignedAgentID = &id
	}
	if recipient := q.Get("recipient"); recipient != "" {
		filter.Recipient = &recipient
	}
	if tag := q.Get("tag"); tag != "" {
		filter.Tag = &tag
	}

	conversations, total, err := h.service.List(r.Context(), tenantID, filter)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondList(w, conversations, Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: (total + perPage - 1) / perPage,
	})
}

func (h *ConversationHandler) GetConversation(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permConversationsRead) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid ID format")
		return
	}
	conversation, err := h.service.GetByID(r.Context(), tenantID, id)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondOK(w, conversation)
}

func (h *ConversationHandler) UpdateConversation(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permConversationsWrite) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid ID format")
		return
	}
	var req ports.UpdateConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}
	if req.AssignedAgentID != nil && !authctx.HasPermission(r.Context(), permConversationsAssign) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	if req.AssignedTeam != nil && !authctx.HasPermission(r.Context(), permConversationsAssign) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	conversation, err := h.service.Update(r.Context(), tenantID, id, req)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondOK(w, conversation)
}

func (h *ConversationHandler) AddNote(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permConversationsWrite) {
		RespondError(w, domain.ErrForbidden)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid ID format")
		return
	}
	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}
	conversation, err := h.service.AddNote(r.Context(), tenantID, id, authctx.UserID(r.Context()), req.Body)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondOK(w, conversation)
}
