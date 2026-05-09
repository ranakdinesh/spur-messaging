package http

import (
	"io"
	"net/http"

	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type WebhookHandler struct {
	service ports.WebhookService
}

func NewWebhookHandler(service ports.WebhookService) *WebhookHandler {
	return &WebhookHandler{service: service}
}

func (h *WebhookHandler) Verify(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	mode := q.Get("hub.mode")
	token := q.Get("hub.verify_token")
	challenge := q.Get("hub.challenge")

	resp, err := h.service.VerifyWhatsAppWebhook(r.Context(), mode, token, challenge)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(resp))
}

func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	h.HandleWhatsApp(w, r)
}

func (h *WebhookHandler) HandleWhatsApp(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = h.service.HandleWhatsAppWebhook(r.Context(), r.Header, body)
	if err != nil {
		// Log error but ALWAYS return 200 to Meta
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) HandleSMS(w http.ResponseWriter, r *http.Request) {
	// SMS webhook handling logic (delivery receipts)
	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) HandleSendGrid(w http.ResponseWriter, r *http.Request) {
	// SendGrid email webhook
	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) HandleMailgun(w http.ResponseWriter, r *http.Request) {
	// Mailgun email webhook
	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) HandlePostmark(w http.ResponseWriter, r *http.Request) {
	// Postmark email webhook
	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) HandleUnsubscribeLink(w http.ResponseWriter, r *http.Request) {
	// HandleUnsubscribeLink public endpoint
	w.WriteHeader(http.StatusOK)
}
