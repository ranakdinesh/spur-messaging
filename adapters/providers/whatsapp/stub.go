package whatsapp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	metaclient "github.com/ranakdinesh/spur-messaging/adapters/providers/whatsapp/meta"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

type MetaClient interface {
	SendTextMessage(ctx context.Context, accessToken, phoneNumberID string, req metaclient.TextMessageRequest) (*metaclient.SendMessageResponse, error)
	SendTemplateMessage(ctx context.Context, accessToken, phoneNumberID string, req metaclient.TemplateMessageRequest) (*metaclient.SendMessageResponse, error)
	SendMediaMessage(ctx context.Context, accessToken, phoneNumberID string, req metaclient.MediaMessageRequest) (*metaclient.SendMessageResponse, error)
	CreateMessageTemplate(ctx context.Context, accessToken, wabaID string, req metaclient.CreateTemplateRequest) (*metaclient.MessageTemplate, error)
	GetTemplateStatus(ctx context.Context, accessToken, templateID string) (*metaclient.MessageTemplate, error)
}

type ProviderOption func(*whatsappProvider)

type whatsappProvider struct {
	client MetaClient
	now    func() time.Time
}

func NewWhatsAppProvider(opts ...ProviderOption) ports.Provider {
	p := &whatsappProvider{
		client: metaclient.NewClient(),
		now:    time.Now,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func WithMetaClient(client MetaClient) ProviderOption {
	return func(p *whatsappProvider) {
		if client != nil {
			p.client = client
		}
	}
}

func (p *whatsappProvider) Channel() domain.Channel {
	return domain.ChannelWhatsApp
}

func (p *whatsappProvider) Send(ctx context.Context, cfg *domain.ProviderConfig, req ports.ProviderSendRequest) (*ports.ProviderSendResult, error) {
	creds, err := whatsappCredentials(cfg)
	if err != nil {
		return nil, err
	}
	if cfg.PhoneNumberID == "" {
		return nil, domain.NewValidationError("phone_number_id", "whatsapp phone number ID is required")
	}

	replyContext := messageContext(req.ReplyToMsgID)
	var resp *metaclient.SendMessageResponse
	switch req.MessageType {
	case domain.MessageTypeText:
		body := ""
		if req.Text != nil {
			body = *req.Text
		}
		if strings.TrimSpace(body) == "" {
			return nil, domain.NewValidationError("text", "text body is required")
		}
		resp, err = p.client.SendTextMessage(ctx, creds.AccessToken, cfg.PhoneNumberID, metaclient.TextMessageRequest{
			To:      req.Recipient,
			Body:    body,
			Context: replyContext,
		})
	case domain.MessageTypeTemplate:
		if req.TemplateName == nil || strings.TrimSpace(*req.TemplateName) == "" {
			return nil, domain.NewValidationError("template_name", "template name is required")
		}
		language := "en"
		if req.TemplateLanguage != nil && strings.TrimSpace(*req.TemplateLanguage) != "" {
			language = *req.TemplateLanguage
		}
		resp, err = p.client.SendTemplateMessage(ctx, creds.AccessToken, cfg.PhoneNumberID, metaclient.TemplateMessageRequest{
			To:         req.Recipient,
			Name:       *req.TemplateName,
			Language:   language,
			Components: templateSendComponents(req.TemplateParams),
			Context:    replyContext,
		})
	case domain.MessageTypeMedia:
		mediaType := ""
		if req.MediaType != nil {
			mediaType = strings.TrimSpace(*req.MediaType)
		}
		mediaRef := ""
		if req.MediaURL != nil {
			mediaRef = strings.TrimSpace(*req.MediaURL)
		}
		if mediaType == "" {
			return nil, domain.NewValidationError("media_type", "media type is required")
		}
		if mediaRef == "" {
			return nil, domain.NewValidationError("media_url", "media URL or media ID is required")
		}
		mediaReq := metaclient.MediaMessageRequest{
			To:      req.Recipient,
			Type:    mediaType,
			Context: replyContext,
		}
		if isURL(mediaRef) {
			mediaReq.Link = mediaRef
		} else {
			mediaReq.MediaID = mediaRef
		}
		if req.Text != nil {
			mediaReq.Caption = *req.Text
		}
		resp, err = p.client.SendMediaMessage(ctx, creds.AccessToken, cfg.PhoneNumberID, mediaReq)
	default:
		return nil, domain.NewValidationError("message_type", "unsupported whatsapp message type")
	}
	if err != nil {
		return nil, mapMetaError(err)
	}

	return &ports.ProviderSendResult{
		ProviderMessageID: firstMessageID(resp),
		Status:            domain.MessageStatusProviderSubmitted,
		Timestamp:         p.now().UTC(),
	}, nil
}

func (p *whatsappProvider) SubmitTemplate(ctx context.Context, cfg *domain.ProviderConfig, tmpl domain.Template) (string, error) {
	creds, err := whatsappCredentials(cfg)
	if err != nil {
		return "", err
	}
	if cfg.WABAID == "" {
		return "", domain.NewValidationError("waba_id", "whatsapp business account ID is required")
	}
	resp, err := p.client.CreateMessageTemplate(ctx, creds.AccessToken, cfg.WABAID, metaclient.CreateTemplateRequest{
		Name:       tmpl.Name,
		Language:   tmpl.Language,
		Category:   string(tmpl.Category),
		Components: templateCreateComponents(tmpl.Components),
	})
	if err != nil {
		return "", mapMetaError(err)
	}
	if resp.ID == "" {
		return "", domain.NewProviderError("meta template creation returned no template ID")
	}
	return resp.ID, nil
}

func (p *whatsappProvider) GetTemplateStatus(ctx context.Context, cfg *domain.ProviderConfig, providerTmplID string) (domain.TemplateStatus, *string, error) {
	creds, err := whatsappCredentials(cfg)
	if err != nil {
		return domain.TemplateStatusPending, nil, err
	}
	resp, err := p.client.GetTemplateStatus(ctx, creds.AccessToken, providerTmplID)
	if err != nil {
		return domain.TemplateStatusPending, nil, mapMetaError(err)
	}
	return mapTemplateStatus(resp.Status), rejectionReason(resp), nil
}

func (p *whatsappProvider) ParseWebhook(ctx context.Context, cfg *domain.ProviderConfig, headers http.Header, body []byte) ([]ports.WebhookEvent, error) {
	if !p.ValidateWebhookSignature(cfg, headers, body) {
		return nil, domain.ErrUnauthorized
	}
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode whatsapp webhook: %w", err)
	}

	events := make([]ports.WebhookEvent, 0)
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}
			for _, status := range change.Value.Statuses {
				events = append(events, statusWebhookEvent(entry.ID, change.Value.Metadata, status))
			}
			for _, msg := range change.Value.Messages {
				events = append(events, inboundWebhookEvent(entry.ID, change.Value.Metadata, msg))
			}
		}
	}
	return events, nil
}

func (p *whatsappProvider) ValidateWebhookSignature(cfg *domain.ProviderConfig, headers http.Header, body []byte) bool {
	creds, err := whatsappCredentials(cfg)
	if err != nil {
		return false
	}
	// Development-only escape hatch: this is false by default and only applies
	// when no Meta app secret is configured. Production configs with an app
	// secret must always provide a valid X-Hub-Signature-256 header.
	if strings.TrimSpace(creds.AppSecret) == "" {
		return creds.WebhookSignatureBypass
	}
	signature := headers.Get("X-Hub-Signature-256")
	if signature == "" {
		return false
	}
	parts := strings.SplitN(signature, "=", 2)
	if len(parts) != 2 || parts[0] != "sha256" || parts[1] == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(creds.AppSecret))
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return subtle.ConstantTimeCompare([]byte(parts[1]), []byte(expected)) == 1
}

func whatsappCredentials(cfg *domain.ProviderConfig) (*domain.WhatsAppCredentials, error) {
	if cfg == nil {
		return nil, domain.ErrProviderNotConfigured
	}
	var creds domain.WhatsAppCredentials
	if err := json.Unmarshal(cfg.Credentials, &creds); err != nil {
		return nil, fmt.Errorf("unmarshal whatsapp credentials: %w", err)
	}
	if strings.TrimSpace(creds.AccessToken) == "" {
		return nil, domain.ErrCredentialsExpired
	}
	return &creds, nil
}

func messageContext(replyTo *string) *metaclient.MessageContext {
	if replyTo == nil || strings.TrimSpace(*replyTo) == "" {
		return nil
	}
	return &metaclient.MessageContext{MessageID: *replyTo}
}

func templateSendComponents(params map[string]string) []metaclient.TemplateComponent {
	if len(params) == 0 {
		return nil
	}
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parameters := make([]metaclient.TemplateParameter, 0, len(keys))
	for _, key := range keys {
		parameters = append(parameters, metaclient.TemplateParameter{Type: "text", Text: params[key]})
	}
	return []metaclient.TemplateComponent{{Type: "body", Parameters: parameters}}
}

func templateCreateComponents(components []domain.TemplateComponent) []metaclient.MessageTemplateComponent {
	out := make([]metaclient.MessageTemplateComponent, 0, len(components))
	for _, component := range components {
		metaComponent := metaclient.MessageTemplateComponent{
			Type:    string(component.Type),
			Buttons: templateButtons(component.Buttons),
		}
		if component.Format != nil {
			metaComponent.Format = *component.Format
		}
		if component.Text != nil {
			metaComponent.Text = *component.Text
		}
		if component.Example != nil {
			metaComponent.Example = &metaclient.TemplateExample{
				HeaderHandle: append([]string(nil), component.Example.HeaderHandle...),
				BodyText:     cloneBodyText(component.Example.BodyText),
			}
		}
		out = append(out, metaComponent)
	}
	return out
}

func templateButtons(buttons []domain.TemplateButton) []metaclient.TemplateButton {
	out := make([]metaclient.TemplateButton, 0, len(buttons))
	for _, button := range buttons {
		metaButton := metaclient.TemplateButton{
			Type: button.Type,
			Text: button.Text,
		}
		if button.URL != nil {
			metaButton.URL = *button.URL
		}
		if button.PhoneNumber != nil {
			metaButton.PhoneNumber = *button.PhoneNumber
		}
		out = append(out, metaButton)
	}
	return out
}

func statusWebhookEvent(wabaID string, metadata WebhookMetadata, status WebhookStatus) ports.WebhookEvent {
	timestamp := parseMetaTimestamp(status.Timestamp)
	mapped := mapWhatsAppStatus(status.Status)
	recipient := status.RecipientID
	event := ports.WebhookEvent{
		Type:               ports.WebhookEventStatusUpdate,
		ProviderMessageID:  status.ID,
		Status:             &mapped,
		Timestamp:          timestamp,
		Recipient:          stringPtr(recipient),
		WABAID:             wabaID,
		PhoneNumberID:      metadata.PhoneNumberID,
		DisplayPhoneNumber: metadata.DisplayPhoneNumber,
		MessageType:        "status",
		Metadata: map[string]string{
			"meta_status":          status.Status,
			"phone_number_id":      metadata.PhoneNumberID,
			"display_phone_number": metadata.DisplayPhoneNumber,
			"recipient_id":         status.RecipientID,
		},
	}
	if len(status.Errors) > 0 {
		errDetail := status.Errors[0]
		event.ErrorCode = strconv.Itoa(errDetail.Code)
		event.ErrorMessage = firstNonEmpty(errDetail.Message, errDetail.Title)
		event.Metadata["error_code"] = event.ErrorCode
		event.Metadata["error_title"] = errDetail.Title
		event.Metadata["error_message"] = errDetail.Message
	}
	return event
}

func inboundWebhookEvent(wabaID string, metadata WebhookMetadata, msg WebhookMessage) ports.WebhookEvent {
	timestamp := parseMetaTimestamp(msg.Timestamp)
	event := ports.WebhookEvent{
		Type:               ports.WebhookEventIncoming,
		ProviderMessageID:  msg.ID,
		Timestamp:          timestamp,
		From:               stringPtr(msg.From),
		Recipient:          stringPtr(metadata.DisplayPhoneNumber),
		WABAID:             wabaID,
		PhoneNumberID:      metadata.PhoneNumberID,
		DisplayPhoneNumber: metadata.DisplayPhoneNumber,
		MessageType:        msg.Type,
		Metadata: map[string]string{
			"message_type":         msg.Type,
			"phone_number_id":      metadata.PhoneNumberID,
			"display_phone_number": metadata.DisplayPhoneNumber,
			"from":                 msg.From,
		},
	}

	switch strings.ToLower(msg.Type) {
	case "text":
		if msg.Text != nil {
			event.Text = stringPtr(msg.Text.Body)
		}
	case "image":
		applyMediaMetadata(&event, msg.Image)
	case "audio":
		applyMediaMetadata(&event, msg.Audio)
	case "video":
		applyMediaMetadata(&event, msg.Video)
	case "document":
		if msg.Document != nil {
			applyMediaMetadata(&event, &WebhookMedia{
				ID:       msg.Document.ID,
				MimeType: msg.Document.MimeType,
				SHA256:   msg.Document.SHA256,
				Caption:  msg.Document.Caption,
			})
			setMeta(event.Metadata, "filename", msg.Document.Filename)
		}
	case "button":
		if msg.Button != nil {
			event.Text = stringPtr(firstNonEmpty(msg.Button.Text, msg.Button.Payload))
			setMeta(event.Metadata, "button_text", msg.Button.Text)
			setMeta(event.Metadata, "button_payload", msg.Button.Payload)
		}
	case "interactive":
		applyInteractiveMetadata(&event, msg.Interactive)
	case "location":
		if msg.Location != nil {
			event.Text = stringPtr(firstNonEmpty(msg.Location.Name, msg.Location.Address))
			event.Metadata["latitude"] = strconv.FormatFloat(msg.Location.Latitude, 'f', -1, 64)
			event.Metadata["longitude"] = strconv.FormatFloat(msg.Location.Longitude, 'f', -1, 64)
			setMeta(event.Metadata, "location_name", msg.Location.Name)
			setMeta(event.Metadata, "location_address", msg.Location.Address)
			setMeta(event.Metadata, "location_url", msg.Location.URL)
		}
	default:
		event.Metadata["unsupported"] = "true"
	}
	if event.Text == nil {
		event.Text = stringPtr("")
	}
	return event
}

func applyMediaMetadata(event *ports.WebhookEvent, media *WebhookMedia) {
	if media == nil {
		return
	}
	event.MediaURL = stringPtr(media.ID)
	setMeta(event.Metadata, "media_id", media.ID)
	setMeta(event.Metadata, "mime_type", media.MimeType)
	setMeta(event.Metadata, "sha256", media.SHA256)
	setMeta(event.Metadata, "caption", media.Caption)
	if media.Caption != "" {
		event.Text = stringPtr(media.Caption)
	}
}

func applyInteractiveMetadata(event *ports.WebhookEvent, interactive *WebhookInteractive) {
	if interactive == nil {
		return
	}
	event.Metadata["interactive_type"] = interactive.Type
	switch strings.ToLower(interactive.Type) {
	case "button_reply":
		if interactive.ButtonReply != nil {
			event.Text = stringPtr(interactive.ButtonReply.Title)
			setMeta(event.Metadata, "reply_id", interactive.ButtonReply.ID)
			setMeta(event.Metadata, "reply_title", interactive.ButtonReply.Title)
			setMeta(event.Metadata, "reply_description", interactive.ButtonReply.Description)
		}
	case "list_reply":
		if interactive.ListReply != nil {
			event.Text = stringPtr(interactive.ListReply.Title)
			setMeta(event.Metadata, "reply_id", interactive.ListReply.ID)
			setMeta(event.Metadata, "reply_title", interactive.ListReply.Title)
			setMeta(event.Metadata, "reply_description", interactive.ListReply.Description)
		}
	default:
		event.Metadata["unsupported_interactive"] = "true"
	}
}

func cloneBodyText(bodyText [][]string) [][]string {
	out := make([][]string, len(bodyText))
	for i := range bodyText {
		out[i] = append([]string(nil), bodyText[i]...)
	}
	return out
}

func mapTemplateStatus(status string) domain.TemplateStatus {
	switch strings.ToUpper(status) {
	case "APPROVED":
		return domain.TemplateStatusApproved
	case "REJECTED":
		return domain.TemplateStatusRejected
	case "PENDING", "IN_REVIEW":
		return domain.TemplateStatusPending
	default:
		return domain.TemplateStatusPending
	}
}

func rejectionReason(tmpl *metaclient.MessageTemplate) *string {
	if tmpl == nil || strings.TrimSpace(tmpl.RejectedReason) == "" {
		return nil
	}
	return &tmpl.RejectedReason
}

func mapWhatsAppStatus(status string) domain.MessageStatus {
	switch strings.ToLower(status) {
	case "sent":
		return domain.MessageStatusSent
	case "delivered":
		return domain.MessageStatusDelivered
	case "read":
		return domain.MessageStatusRead
	case "failed":
		return domain.MessageStatusFailed
	case "deleted":
		return domain.MessageStatusCancelled
	case "expired":
		return domain.MessageStatusExpired
	default:
		return domain.MessageStatusProviderSubmitted
	}
}

func parseMetaTimestamp(raw string) time.Time {
	if unix, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return time.Unix(unix, 0).UTC()
	}
	return time.Now().UTC()
}

func firstMessageID(resp *metaclient.SendMessageResponse) string {
	if resp == nil || len(resp.Messages) == 0 {
		return ""
	}
	return resp.Messages[0].ID
}

func isURL(value string) bool {
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

func stringPtr(value string) *string {
	return &value
}

func setMeta(metadata map[string]string, key, value string) {
	if value != "" {
		metadata[key] = value
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func mapMetaError(err error) error {
	var httpErr *metaclient.HTTPError
	if !errors.As(err, &httpErr) || httpErr.MetaError == nil {
		return err
	}
	metaErr := httpErr.MetaError
	switch {
	case metaErr.IsRateLimit():
		return domain.ErrRateLimitExceeded
	case metaErr.IsAuthError():
		return domain.ErrCredentialsExpired
	case metaErr.IsPermissionError():
		return domain.ErrForbidden
	case metaErr.IsTemplateError():
		return domain.NewProviderError(metaErr.SafeMessage())
	case metaErr.IsTemporary():
		return domain.ErrProviderTimeout
	default:
		return domain.NewProviderError(metaErr.SafeMessage())
	}
}
