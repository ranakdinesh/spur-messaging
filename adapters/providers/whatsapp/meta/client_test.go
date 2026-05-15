package meta

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientSendTextUsesConfigurableVersionAndBearerToken(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"messaging_product":"whatsapp","messages":[{"id":"wamid.123"}]}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithAPIVersion("v99.0"))
	resp, err := client.SendTextMessage(context.Background(), "secret-token", "phone-123", TextMessageRequest{
		To:   "+15551234567",
		Body: "hello",
	})
	if err != nil {
		t.Fatalf("SendTextMessage returned error: %v", err)
	}
	if client.APIVersion() != "v99.0" {
		t.Fatalf("API version = %q, want v99.0", client.APIVersion())
	}
	if gotPath != "/v99.0/phone-123/messages" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAuth != "Bearer secret-token" {
		t.Fatalf("authorization header was not set")
	}
	if gotPayload["messaging_product"] != "whatsapp" || gotPayload["type"] != "text" {
		t.Fatalf("unexpected payload: %#v", gotPayload)
	}
	if len(resp.Messages) != 1 || resp.Messages[0].ID != "wamid.123" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestClientSendTemplateMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v23.0/phone-123/messages" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload["type"] != "template" {
			t.Fatalf("type = %#v", payload["type"])
		}
		template := payload["template"].(map[string]any)
		if template["name"] != "order_update" {
			t.Fatalf("template name = %#v", template["name"])
		}
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.template"}]}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	_, err := client.SendTemplateMessage(context.Background(), "token", "phone-123", TemplateMessageRequest{
		To:       "+15551234567",
		Name:     "order_update",
		Language: "en_US",
		Components: []TemplateComponent{{
			Type: "body",
			Parameters: []TemplateParameter{{
				Type: "text",
				Text: "A123",
			}},
		}},
	})
	if err != nil {
		t.Fatalf("SendTemplateMessage returned error: %v", err)
	}
}

func TestClientSendMediaMessageByIDAndURL(t *testing.T) {
	var seen []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		seen = append(seen, payload)
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.media"}]}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	if _, err := client.SendMediaMessage(context.Background(), "token", "phone-123", MediaMessageRequest{
		To:      "+15551234567",
		Type:    "image",
		MediaID: "media-id",
	}); err != nil {
		t.Fatalf("SendMediaMessage by id returned error: %v", err)
	}
	if _, err := client.SendMediaMessage(context.Background(), "token", "phone-123", MediaMessageRequest{
		To:   "+15551234567",
		Type: "document",
		Link: "https://example.com/file.pdf",
	}); err != nil {
		t.Fatalf("SendMediaMessage by link returned error: %v", err)
	}
	if seen[0]["image"].(map[string]any)["id"] != "media-id" {
		t.Fatalf("image media id missing: %#v", seen[0])
	}
	if seen[1]["document"].(map[string]any)["link"] != "https://example.com/file.pdf" {
		t.Fatalf("document link missing: %#v", seen[1])
	}
}

func TestClientUploadMediaUsesMultipart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Fatalf("content type = %q", r.Header.Get("Content-Type"))
		}
		if err := r.ParseMultipartForm(1024 * 1024); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		if r.FormValue("messaging_product") != "whatsapp" || r.FormValue("type") != "image/png" {
			t.Fatalf("unexpected form values")
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("form file: %v", err)
		}
		defer file.Close()
		contents, _ := io.ReadAll(file)
		if string(contents) != "png-bytes" {
			t.Fatalf("file contents = %q", string(contents))
		}
		_, _ = w.Write([]byte(`{"id":"media-123"}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	resp, err := client.UploadMedia(context.Background(), "token", "phone-123", "image/png", "image.png", strings.NewReader("png-bytes"))
	if err != nil {
		t.Fatalf("UploadMedia returned error: %v", err)
	}
	if resp.ID != "media-123" {
		t.Fatalf("media id = %q", resp.ID)
	}
}

func TestClientTemplateAndAccountEndpoints(t *testing.T) {
	paths := make([]string, 0, 7)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch {
		case strings.HasSuffix(r.URL.Path, "/message_templates") && r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"id":"tmpl-123","name":"welcome","status":"PENDING"}`))
		case strings.HasSuffix(r.URL.Path, "/message_templates") && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":[{"id":"tmpl-123","name":"welcome","status":"APPROVED"}]}`))
		case strings.Contains(r.URL.Path, "tmpl-123"):
			_, _ = w.Write([]byte(`{"id":"tmpl-123","name":"welcome","status":"APPROVED"}`))
		case strings.Contains(r.URL.Path, "phone_numbers"):
			_, _ = w.Write([]byte(`{"data":[{"id":"phone-123","display_phone_number":"+1 555 123 4567"}]}`))
		case strings.Contains(r.URL.Path, "phone-123"):
			_, _ = w.Write([]byte(`{"id":"phone-123","display_phone_number":"+1 555 123 4567","quality_rating":"GREEN"}`))
		default:
			_, _ = w.Write([]byte(`{"id":"waba-123","name":"Acme","currency":"USD","timezone_id":"1"}`))
		}
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	if _, err := client.CreateMessageTemplate(context.Background(), "token", "waba-123", CreateTemplateRequest{Name: "welcome", Language: "en_US", Category: "UTILITY"}); err != nil {
		t.Fatalf("CreateMessageTemplate returned error: %v", err)
	}
	if _, err := client.ListTemplates(context.Background(), "token", "waba-123"); err != nil {
		t.Fatalf("ListTemplates returned error: %v", err)
	}
	if _, err := client.GetTemplateStatus(context.Background(), "token", "tmpl-123"); err != nil {
		t.Fatalf("GetTemplateStatus returned error: %v", err)
	}
	if _, err := client.GetWABA(context.Background(), "token", "waba-123"); err != nil {
		t.Fatalf("GetWABA returned error: %v", err)
	}
	if _, err := client.ListPhoneNumbers(context.Background(), "token", "waba-123"); err != nil {
		t.Fatalf("ListPhoneNumbers returned error: %v", err)
	}
	if _, err := client.GetPhoneNumber(context.Background(), "token", "phone-123"); err != nil {
		t.Fatalf("GetPhoneNumber returned error: %v", err)
	}
	if len(paths) != 6 {
		t.Fatalf("paths len = %d, paths = %#v", len(paths), paths)
	}
}

func TestClientRegisterAndVerifyPhoneNumber(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	registered, err := client.RegisterPhoneNumber(context.Background(), "token", "phone-123", "123456")
	if err != nil {
		t.Fatalf("RegisterPhoneNumber returned error: %v", err)
	}
	verified, err := client.VerifyPhoneNumberCode(context.Background(), "token", "phone-123", "654321")
	if err != nil {
		t.Fatalf("VerifyPhoneNumberCode returned error: %v", err)
	}
	if !registered.OK() || !verified.OK() {
		t.Fatalf("success response not recognized")
	}
	if paths[0] != "/v23.0/phone-123/register" || paths[1] != "/v23.0/phone-123/verify_code" {
		t.Fatalf("unexpected paths: %#v", paths)
	}
}

func TestClientDecodesMetaErrorWithoutToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"Invalid parameter","type":"OAuthException","code":100,"error_subcode":2388024,"fbtrace_id":"trace-123"}}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	_, err := client.SendTextMessage(context.Background(), "super-secret-token", "phone-123", TextMessageRequest{To: "+1", Body: "hello"})
	if err == nil {
		t.Fatal("expected error")
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.MetaError == nil || httpErr.MetaError.Code != 100 || httpErr.MetaError.ErrorSubcode != 2388024 {
		t.Fatalf("unexpected meta error: %#v", httpErr.MetaError)
	}
	if strings.Contains(err.Error(), "super-secret-token") {
		t.Fatalf("error leaked access token: %v", err)
	}
}

func TestClientContextCancellationIsRespected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.SendTextMessage(ctx, "token", "phone-123", TextMessageRequest{To: "+1", Body: "hello"})
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestClientTimeoutOption(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithTimeout(1*time.Millisecond))
	_, err := client.SendTextMessage(context.Background(), "token", "phone-123", TextMessageRequest{To: "+1", Body: "hello"})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
