package whatsapp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	metaclient "github.com/ranakdinesh/spur-messaging/adapters/providers/whatsapp/meta"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

func TestWhatsAppProviderSendText(t *testing.T) {
	var gotPath string
	var gotPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if strings.Contains(r.Header.Get("Authorization"), "test-token") == false {
			t.Fatalf("authorization header missing")
		}
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.text"}]}`))
	}))
	defer server.Close()

	provider := newTestProvider(server.URL)
	text := "hello"
	result, err := provider.Send(context.Background(), testConfig(), ports.ProviderSendRequest{
		Recipient:   "+15551234567",
		MessageType: domain.MessageTypeText,
		Text:        &text,
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if gotPath != "/v99.0/phone-123/messages" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotPayload["type"] != "text" {
		t.Fatalf("payload type = %#v", gotPayload["type"])
	}
	if result.ProviderMessageID != "wamid.text" {
		t.Fatalf("provider message id = %q", result.ProviderMessageID)
	}
	if result.Status != domain.MessageStatusProviderSubmitted {
		t.Fatalf("status = %q", result.Status)
	}
}

func TestWhatsAppProviderSendTemplate(t *testing.T) {
	var gotPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.template"}]}`))
	}))
	defer server.Close()

	provider := newTestProvider(server.URL)
	name := "order_update"
	lang := "en_US"
	result, err := provider.Send(context.Background(), testConfig(), ports.ProviderSendRequest{
		Recipient:        "+15551234567",
		MessageType:      domain.MessageTypeTemplate,
		TemplateName:     &name,
		TemplateLanguage: &lang,
		TemplateParams: map[string]string{
			"2": "delivered",
			"1": "A123",
		},
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	template := gotPayload["template"].(map[string]any)
	if template["name"] != "order_update" {
		t.Fatalf("template name = %#v", template["name"])
	}
	components := template["components"].([]any)
	parameters := components[0].(map[string]any)["parameters"].([]any)
	if parameters[0].(map[string]any)["text"] != "A123" || parameters[1].(map[string]any)["text"] != "delivered" {
		t.Fatalf("template parameters not sorted by key: %#v", parameters)
	}
	if result.ProviderMessageID != "wamid.template" {
		t.Fatalf("provider message id = %q", result.ProviderMessageID)
	}
}

func TestWhatsAppProviderSendMediaByURL(t *testing.T) {
	var gotPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.media"}]}`))
	}))
	defer server.Close()

	provider := newTestProvider(server.URL)
	mediaURL := "https://example.com/image.png"
	mediaType := "image"
	caption := "caption"
	_, err := provider.Send(context.Background(), testConfig(), ports.ProviderSendRequest{
		Recipient:   "+15551234567",
		MessageType: domain.MessageTypeMedia,
		MediaURL:    &mediaURL,
		MediaType:   &mediaType,
		Text:        &caption,
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	image := gotPayload["image"].(map[string]any)
	if image["link"] != mediaURL || image["caption"] != caption {
		t.Fatalf("unexpected image payload: %#v", image)
	}
}

func TestWhatsAppProviderSubmitTemplate(t *testing.T) {
	var gotPath string
	var gotPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"id":"tmpl-123","status":"PENDING"}`))
	}))
	defer server.Close()

	provider := newTestProvider(server.URL)
	body := "Hello {{1}}"
	id, err := provider.SubmitTemplate(context.Background(), testConfig(), domain.Template{
		Name:     "hello_world",
		Language: "en_US",
		Category: domain.TemplateCategoryUtility,
		Components: []domain.TemplateComponent{{
			Type: domain.ComponentBody,
			Text: &body,
		}},
	})
	if err != nil {
		t.Fatalf("SubmitTemplate returned error: %v", err)
	}
	if gotPath != "/v99.0/waba-123/message_templates" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotPayload["name"] != "hello_world" {
		t.Fatalf("template payload = %#v", gotPayload)
	}
	if id != "tmpl-123" {
		t.Fatalf("template id = %q", id)
	}
}

func TestWhatsAppProviderGetTemplateStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"tmpl-123","status":"REJECTED","rejected_reason":"bad sample"}`))
	}))
	defer server.Close()

	provider := newTestProvider(server.URL)
	status, reason, err := provider.GetTemplateStatus(context.Background(), testConfig(), "tmpl-123")
	if err != nil {
		t.Fatalf("GetTemplateStatus returned error: %v", err)
	}
	if status != domain.TemplateStatusRejected {
		t.Fatalf("status = %q", status)
	}
	if reason == nil || *reason != "bad sample" {
		t.Fatalf("reason = %#v", reason)
	}
}

func TestWhatsAppProviderMapsMetaErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"Application request limit reached","type":"OAuthException","code":4,"fbtrace_id":"trace-rate"}}`))
	}))
	defer server.Close()

	provider := newTestProvider(server.URL)
	text := "hello"
	_, err := provider.Send(context.Background(), testConfig(), ports.ProviderSendRequest{
		Recipient:   "+15551234567",
		MessageType: domain.MessageTypeText,
		Text:        &text,
	})
	if !errors.Is(err, domain.ErrRateLimitExceeded) {
		t.Fatalf("error = %v, want rate limit", err)
	}
	if strings.Contains(err.Error(), "test-token") {
		t.Fatalf("error leaked token: %v", err)
	}
}

func TestWhatsAppProviderParseWebhookAndValidateSignature(t *testing.T) {
	body := []byte(`{"object":"whatsapp_business_account","entry":[{"id":"waba-123","changes":[{"field":"messages","value":{"messaging_product":"whatsapp","metadata":{"display_phone_number":"+1 555 000 1111","phone_number_id":"phone-123"},"statuses":[{"id":"wamid.1","status":"delivered","timestamp":"1700000000","recipient_id":"15551234567"}],"messages":[{"from":"15551234567","id":"wamid.in","timestamp":"1700000001","type":"text","text":{"body":"STOP"}}]}}]}]}`)
	headers := http.Header{}
	headers.Set("X-Hub-Signature-256", signatureFor("app-secret", body))

	provider := newTestProvider("http://example.invalid")
	events, err := provider.ParseWebhook(context.Background(), testConfig(), headers, body)
	if err != nil {
		t.Fatalf("ParseWebhook returned error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("events len = %d", len(events))
	}
	if events[0].Type != ports.WebhookEventStatusUpdate || events[0].Status == nil || *events[0].Status != domain.MessageStatusDelivered {
		t.Fatalf("unexpected status event: %#v", events[0])
	}
	if events[0].WABAID != "waba-123" || events[0].PhoneNumberID != "phone-123" || events[0].DisplayPhoneNumber != "+1 555 000 1111" {
		t.Fatalf("metadata not preserved: %#v", events[0])
	}
	if events[1].Type != ports.WebhookEventIncoming || events[1].Text == nil || *events[1].Text != "STOP" {
		t.Fatalf("unexpected inbound event: %#v", events[1])
	}
	if provider.ValidateWebhookSignature(testConfig(), http.Header{}, body) {
		t.Fatalf("missing signature validated")
	}
}

func TestWhatsAppProviderValidateWebhookSignature(t *testing.T) {
	body := []byte(`{"entry":[]}`)
	validHeaders := signedHeaders(body)
	invalidHeaders := http.Header{}
	invalidHeaders.Set("X-Hub-Signature-256", signatureFor("wrong-secret", body))
	malformedHeaders := http.Header{}
	malformedHeaders.Set("X-Hub-Signature-256", "sha256-not-valid")

	tests := []struct {
		name    string
		cfg     *domain.ProviderConfig
		headers http.Header
		want    bool
	}{
		{
			name:    "valid signature accepted",
			cfg:     testConfig(),
			headers: validHeaders,
			want:    true,
		},
		{
			name:    "invalid signature rejected",
			cfg:     testConfig(),
			headers: invalidHeaders,
			want:    false,
		},
		{
			name:    "missing signature rejected when app secret configured",
			cfg:     testConfig(),
			headers: http.Header{},
			want:    false,
		},
		{
			name:    "malformed signature rejected",
			cfg:     testConfig(),
			headers: malformedHeaders,
			want:    false,
		},
		{
			name:    "empty app secret rejects by default",
			cfg:     testConfigWithCredentials(domain.WhatsAppCredentials{AccessToken: "test-token"}),
			headers: http.Header{},
			want:    false,
		},
		{
			name: "empty app secret allows explicit development bypass",
			cfg: testConfigWithCredentials(domain.WhatsAppCredentials{
				AccessToken:            "test-token",
				WebhookSignatureBypass: true,
			}),
			headers: http.Header{},
			want:    true,
		},
		{
			name: "development bypass does not override configured app secret",
			cfg: testConfigWithCredentials(domain.WhatsAppCredentials{
				AccessToken:            "test-token",
				AppSecret:              "app-secret",
				WebhookSignatureBypass: true,
			}),
			headers: http.Header{},
			want:    false,
		},
	}

	provider := newTestProvider("http://example.invalid")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := provider.ValidateWebhookSignature(tt.cfg, tt.headers, body); got != tt.want {
				t.Fatalf("ValidateWebhookSignature = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWhatsAppProviderParseStatusPayloads(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		wantStatus domain.MessageStatus
	}{
		{name: "sent", status: "sent", wantStatus: domain.MessageStatusSent},
		{name: "delivered", status: "delivered", wantStatus: domain.MessageStatusDelivered},
		{name: "read", status: "read", wantStatus: domain.MessageStatusRead},
		{name: "failed", status: "failed", wantStatus: domain.MessageStatusFailed},
		{name: "deleted", status: "deleted", wantStatus: domain.MessageStatusCancelled},
		{name: "expired", status: "expired", wantStatus: domain.MessageStatusExpired},
	}

	provider := newTestProvider("http://example.invalid")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := []byte(`{"entry":[{"id":"waba-123","changes":[{"field":"messages","value":{"metadata":{"display_phone_number":"+1 555 000 1111","phone_number_id":"phone-123"},"statuses":[{"id":"wamid.status","status":"` + tt.status + `","timestamp":"1700000000","recipient_id":"15551234567"}]}}]}]}`)
			events, err := provider.ParseWebhook(context.Background(), testConfig(), signedHeaders(body), body)
			if err != nil {
				t.Fatalf("ParseWebhook returned error: %v", err)
			}
			if len(events) != 1 {
				t.Fatalf("events len = %d", len(events))
			}
			if events[0].Status == nil || *events[0].Status != tt.wantStatus {
				t.Fatalf("status = %#v, want %s", events[0].Status, tt.wantStatus)
			}
			if events[0].Metadata["meta_status"] != tt.status {
				t.Fatalf("meta status not preserved: %#v", events[0].Metadata)
			}
		})
	}
}

func TestWhatsAppProviderParseFailedStatusIncludesErrorDetail(t *testing.T) {
	body := []byte(`{"entry":[{"id":"waba-123","changes":[{"field":"messages","value":{"metadata":{"display_phone_number":"+1 555 000 1111","phone_number_id":"phone-123"},"statuses":[{"id":"wamid.failed","status":"failed","timestamp":"1700000000","recipient_id":"15551234567","errors":[{"code":131026,"title":"Message undeliverable","message":"Recipient is unavailable"}]}]}}]}]}`)
	provider := newTestProvider("http://example.invalid")
	events, err := provider.ParseWebhook(context.Background(), testConfig(), signedHeaders(body), body)
	if err != nil {
		t.Fatalf("ParseWebhook returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events len = %d", len(events))
	}
	if events[0].ErrorCode != "131026" || events[0].ErrorMessage != "Recipient is unavailable" {
		t.Fatalf("error detail not preserved: %#v", events[0])
	}
}

func TestWhatsAppProviderParseInboundPayloads(t *testing.T) {
	tests := []struct {
		name      string
		message   string
		wantText  string
		wantMedia string
		wantMeta  map[string]string
	}{
		{
			name:     "text",
			message:  `{"from":"15551234567","id":"wamid.text","timestamp":"1700000001","type":"text","text":{"body":"hello"}}`,
			wantText: "hello",
		},
		{
			name:      "image",
			message:   `{"from":"15551234567","id":"wamid.image","timestamp":"1700000001","type":"image","image":{"id":"media-image","mime_type":"image/jpeg","sha256":"abc","caption":"photo"}}`,
			wantText:  "photo",
			wantMedia: "media-image",
			wantMeta:  map[string]string{"mime_type": "image/jpeg"},
		},
		{
			name:      "document",
			message:   `{"from":"15551234567","id":"wamid.doc","timestamp":"1700000001","type":"document","document":{"id":"media-doc","mime_type":"application/pdf","filename":"file.pdf","caption":"invoice"}}`,
			wantText:  "invoice",
			wantMedia: "media-doc",
			wantMeta:  map[string]string{"filename": "file.pdf"},
		},
		{
			name:      "audio",
			message:   `{"from":"15551234567","id":"wamid.audio","timestamp":"1700000001","type":"audio","audio":{"id":"media-audio","mime_type":"audio/ogg"}}`,
			wantMedia: "media-audio",
		},
		{
			name:      "video",
			message:   `{"from":"15551234567","id":"wamid.video","timestamp":"1700000001","type":"video","video":{"id":"media-video","mime_type":"video/mp4","caption":"clip"}}`,
			wantText:  "clip",
			wantMedia: "media-video",
		},
		{
			name:     "button",
			message:  `{"from":"15551234567","id":"wamid.button","timestamp":"1700000001","type":"button","button":{"text":"Yes","payload":"YES_PAYLOAD"}}`,
			wantText: "Yes",
			wantMeta: map[string]string{"button_payload": "YES_PAYLOAD"},
		},
		{
			name:     "interactive button reply",
			message:  `{"from":"15551234567","id":"wamid.ibutton","timestamp":"1700000001","type":"interactive","interactive":{"type":"button_reply","button_reply":{"id":"yes","title":"Yes"}}}`,
			wantText: "Yes",
			wantMeta: map[string]string{"interactive_type": "button_reply", "reply_id": "yes"},
		},
		{
			name:     "interactive list reply",
			message:  `{"from":"15551234567","id":"wamid.list","timestamp":"1700000001","type":"interactive","interactive":{"type":"list_reply","list_reply":{"id":"plan_a","title":"Plan A","description":"Best plan"}}}`,
			wantText: "Plan A",
			wantMeta: map[string]string{"interactive_type": "list_reply", "reply_description": "Best plan"},
		},
		{
			name:     "location",
			message:  `{"from":"15551234567","id":"wamid.loc","timestamp":"1700000001","type":"location","location":{"latitude":12.34,"longitude":56.78,"name":"Office","address":"Main Road"}}`,
			wantText: "Office",
			wantMeta: map[string]string{"latitude": "12.34", "longitude": "56.78", "location_address": "Main Road"},
		},
		{
			name:     "unsupported",
			message:  `{"from":"15551234567","id":"wamid.unknown","timestamp":"1700000001","type":"contacts","contacts":[{"name":{"formatted_name":"A"}}]}`,
			wantText: "",
			wantMeta: map[string]string{"unsupported": "true", "message_type": "contacts"},
		},
	}

	provider := newTestProvider("http://example.invalid")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := []byte(`{"entry":[{"id":"waba-123","changes":[{"field":"messages","value":{"metadata":{"display_phone_number":"+1 555 000 1111","phone_number_id":"phone-123"},"messages":[` + tt.message + `]}}]}]}`)
			events, err := provider.ParseWebhook(context.Background(), testConfig(), signedHeaders(body), body)
			if err != nil {
				t.Fatalf("ParseWebhook returned error: %v", err)
			}
			if len(events) != 1 {
				t.Fatalf("events len = %d", len(events))
			}
			event := events[0]
			if event.Type != ports.WebhookEventIncoming {
				t.Fatalf("event type = %s", event.Type)
			}
			if event.WABAID != "waba-123" || event.PhoneNumberID != "phone-123" || event.DisplayPhoneNumber != "+1 555 000 1111" {
				t.Fatalf("routing metadata not preserved: %#v", event)
			}
			if event.Text == nil || *event.Text != tt.wantText {
				t.Fatalf("text = %#v, want %q", event.Text, tt.wantText)
			}
			if tt.wantMedia != "" && (event.MediaURL == nil || *event.MediaURL != tt.wantMedia) {
				t.Fatalf("media = %#v, want %q", event.MediaURL, tt.wantMedia)
			}
			for key, want := range tt.wantMeta {
				if event.Metadata[key] != want {
					t.Fatalf("metadata[%s] = %q, want %q; metadata=%#v", key, event.Metadata[key], want, event.Metadata)
				}
			}
		})
	}
}

func TestWhatsAppProviderParseDuplicateAndMalformedPayloadsSafely(t *testing.T) {
	provider := newTestProvider("http://example.invalid")
	body := []byte(`{"entry":[{"id":"waba-123","changes":[{"field":"messages","value":{"metadata":{"phone_number_id":"phone-123"},"messages":[{"from":"15551234567","id":"wamid.same","timestamp":"1700000001","type":"text","text":{"body":"one"}},{"from":"15551234567","id":"wamid.same","timestamp":"1700000001","type":"text","text":{"body":"one"}}]}}]}]}`)
	events, err := provider.ParseWebhook(context.Background(), testConfig(), signedHeaders(body), body)
	if err != nil {
		t.Fatalf("ParseWebhook returned error for duplicate payload: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("duplicate payload should be represented safely, len = %d", len(events))
	}

	malformed := []byte(`{"entry":[`)
	if _, err := provider.ParseWebhook(context.Background(), testConfig(), signedHeaders(malformed), malformed); err == nil {
		t.Fatalf("expected malformed JSON error")
	}
}

func TestWhatsAppProviderDoesNotReturnNotImplemented(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.text"}]}`))
	}))
	defer server.Close()

	provider := newTestProvider(server.URL)
	text := "hello"
	_, err := provider.Send(context.Background(), testConfig(), ports.ProviderSendRequest{
		Recipient:   "+15551234567",
		MessageType: domain.MessageTypeText,
		Text:        &text,
	})
	if err != nil && strings.Contains(err.Error(), "fully implemented") {
		t.Fatalf("provider returned the old stub marker: %v", err)
	}
}

func newTestProvider(baseURL string) ports.Provider {
	return NewWhatsAppProvider(WithMetaClient(metaclient.NewClient(
		metaclient.WithBaseURL(baseURL),
		metaclient.WithAPIVersion("v99.0"),
	)))
}

func testConfig() *domain.ProviderConfig {
	return testConfigWithCredentials(domain.WhatsAppCredentials{
		AccessToken: "test-token",
		AppSecret:   "app-secret",
	})
}

func testConfigWithCredentials(credentials domain.WhatsAppCredentials) *domain.ProviderConfig {
	creds, _ := json.Marshal(credentials)
	return &domain.ProviderConfig{
		Channel:       domain.ChannelWhatsApp,
		Provider:      "meta_cloud",
		Credentials:   creds,
		PhoneNumberID: "phone-123",
		WABAID:        "waba-123",
	}
}

func signatureFor(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func signedHeaders(body []byte) http.Header {
	headers := http.Header{}
	headers.Set("X-Hub-Signature-256", signatureFor("app-secret", body))
	return headers
}
