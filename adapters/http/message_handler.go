package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
	"github.com/ranakdinesh/spur-messaging/pkg/authctx"
)

type MessageHandler struct {
	service ports.MessageService
}

func NewMessageHandler(service ports.MessageService) *MessageHandler {
	return &MessageHandler{service: service}
}

func (h *MessageHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	tenantID := authctx.TenantID(r.Context())
	if !authctx.HasPermission(r.Context(), permMessagesSend) {
		RespondError(w, domain.ErrForbidden)
		return
	}

	var body struct {
		ports.SendMessageRequest
		TrackOpens  *bool    `json:"track_opens"`
		TrackClicks *bool    `json:"track_clicks"`
		FromEmail   string   `json:"from_email"`
		FromName    string   `json:"from_name"`
		ReplyTo     string   `json:"reply_to"`
		SenderID    string   `json:"sender_id"`
		Idempotency string   `json:"idempotency_key"`
		CC          []string `json:"cc"`
		BCC         []string `json:"bcc"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	req := body.SendMessageRequest
	req.TrackOpens = body.TrackOpens
	req.TrackClicks = body.TrackClicks
	req.FromEmail = body.FromEmail
	req.FromName = body.FromName
	req.ReplyTo = body.ReplyTo
	req.SenderID = body.SenderID
	req.CC = body.CC
	req.BCC = body.BCC

	headerKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	bodyKey := strings.TrimSpace(body.Idempotency)
	if headerKey != "" && bodyKey != "" && headerKey != bodyKey {
		RespondError(w, domain.NewValidationError("idempotency_key", "Idempotency-Key header and idempotency_key body field must match"))
		return
	}
	req.IdempotencyKey = headerKey
	if req.IdempotencyKey == "" {
		req.IdempotencyKey = bodyKey
	}
	if err := validateIdempotencyKey(req.IdempotencyKey); err != nil {
		RespondError(w, err)
		return
	}

	if err := validateChannel(req.Channel); err != nil {
		RespondError(w, err)
		return
	}

	if err := validateMetadata(req.Metadata); err != nil {
		RespondError(w, err)
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
	if !authctx.HasPermission(r.Context(), permMessagesSendBulk) {
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
	if !authctx.HasPermission(r.Context(), permMessagesRead) {
		RespondError(w, domain.ErrForbidden)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondValidationError(w, "id", "invalid ID format")
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
	if !authctx.HasPermission(r.Context(), permMessagesRead) {
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
		if !domain.IsValidMessageStatus(s) {
			RespondValidationError(w, "status", "invalid message status")
			return
		}
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
