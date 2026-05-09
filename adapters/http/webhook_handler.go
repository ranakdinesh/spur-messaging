package http

import (
	"io"
	"net/http"

	"github.com/ranakdinesh/spur-messaging/core/ports"
)

// WebhookConfig contains platform-level webhook configuration to avoid cyclic import
type WebhookConfig struct {
	WhatsAppWebhookVerifyToken string
}

// Logger is a subset of the platform logger to avoid cyclic import
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type WebhookHandler struct {
	service ports.WebhookService
	cfg     WebhookConfig
	log     Logger
}

func NewWebhookHandler(service ports.WebhookService, cfg WebhookConfig, log Logger) *WebhookHandler {
	return &WebhookHandler{
		service: service,
		cfg:     cfg,
		log:     log,
	}
}

func (h *WebhookHandler) Verify(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	if mode == "subscribe" && token == h.cfg.WhatsAppWebhookVerifyToken {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(challenge))
		return
	}
	w.WriteHeader(http.StatusForbidden)
}

func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	h.HandleWhatsApp(w, r)
}

func (h *WebhookHandler) HandleWhatsApp(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.Error("failed to read webhook body", "error", err)
		w.WriteHeader(http.StatusOK) // Meta expects 200
		return
	}

	// ALWAYS return 200 OK to Meta regardless of processing errors
	w.WriteHeader(http.StatusOK)

	err = h.service.HandleWhatsAppWebhook(r.Context(), r.Header, body)
	if err != nil {
		h.log.Error("failed to handle whatsapp webhook", "error", err)
	}
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
