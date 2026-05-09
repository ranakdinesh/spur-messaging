package http

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
	"github.com/ranakdinesh/spur-messaging/pkg/authctx"
)

var (
	phoneRegex = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

func validatePhone(phone string) (string, error) {
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")
	if !phoneRegex.MatchString(phone) {
		return "", domain.NewValidationError("phone", "phone must be E.164 format (e.g. +919810914244)")
	}
	return phone, nil
}

func validateEmail(email string) (string, error) {
	if len(email) > 254 {
		return "", domain.NewValidationError("email", "invalid email address")
	}
	if !emailRegex.MatchString(email) {
		return "", domain.NewValidationError("email", "invalid email address")
	}
	return strings.ToLower(email), nil
}

func validateTags(tags []string) error {
	if len(tags) > 10 {
		return domain.NewValidationError("tags", "max 10 tags allowed, each max 50 chars")
	}
	for _, t := range tags {
		if len(t) > 50 {
			return domain.NewValidationError("tags", "max 10 tags allowed, each max 50 chars")
		}
	}
	return nil
}

func validateMetadata(metadata map[string]string) error {
	if len(metadata) > 20 {
		return domain.NewValidationError("metadata", "metadata: max 20 keys, key max 50, value max 500 chars")
	}
	for k, v := range metadata {
		if len(k) > 50 || len(v) > 500 {
			return domain.NewValidationError("metadata", "metadata: max 20 keys, key max 50, value max 500 chars")
		}
	}
	return nil
}

type MessageHandler struct {
	service ports.MessageService
}

func NewMessageHandler(service ports.MessageService) *MessageHandler {
	return &MessageHandler{service: service}
}

func (h *MessageHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:messages:send") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	var req ports.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	if req.Channel == domain.ChannelEmail {
		email, err := validateEmail(req.Recipient)
		if err != nil {
			RespondError(w, err)
			return
		}
		req.Recipient = email
	} else {
		phone, err := validatePhone(req.Recipient)
		if err != nil {
			RespondError(w, err)
			return
		}
		req.Recipient = phone
	}

	if err := validateMetadata(req.Metadata); err != nil {
		RespondError(w, err)
		return
	}

	msg, err := h.service.Send(r.Context(), tenantID, req)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondCreated(w, msg)
}

func (h *MessageHandler) SendBulkMessages(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:messages:send_bulk") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	var reqs []ports.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	for i := range reqs {
		if reqs[i].Channel == domain.ChannelEmail {
			email, err := validateEmail(reqs[i].Recipient)
			if err != nil {
				RespondError(w, err)
				return
			}
			reqs[i].Recipient = email
		} else {
			phone, err := validatePhone(reqs[i].Recipient)
			if err != nil {
				RespondError(w, err)
				return
			}
			reqs[i].Recipient = phone
		}
		if err := validateMetadata(reqs[i].Metadata); err != nil {
			RespondError(w, err)
			return
		}
	}

	msgs, err := h.service.SendBulk(r.Context(), tenantID, reqs)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondCreated(w, msgs)
}

func (h *MessageHandler) GetMessage(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:messages:read") {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid message ID")
		return
	}

	msg, err := h.service.GetByID(r.Context(), tenantID, id)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondOK(w, msg)
}

func (h *MessageHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), "messaging:messages:read") {
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

	filter := ports.MessageFilter{
		Page:    page,
		PerPage: perPage,
	}

	if channel := q.Get("channel"); channel != "" {
		c := domain.Channel(channel)
		filter.Channel = &c
	}
	if status := q.Get("status"); status != "" {
		s := domain.MessageStatus(status)
		filter.Status = &s
	}
	if recipient := q.Get("recipient"); recipient != "" {
		filter.Recipient = &recipient
	}

	msgs, total, err := h.service.List(r.Context(), tenantID, filter)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondList(w, msgs, Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: (total + perPage - 1) / perPage,
	})
}
