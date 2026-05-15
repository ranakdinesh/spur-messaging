package meta

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetaErrorClassification(t *testing.T) {
	tests := []struct {
		name       string
		err        MetaError
		rateLimit  bool
		auth       bool
		permission bool
		template   bool
		temporary  bool
	}{
		{
			name:      "rate limit code",
			err:       MetaError{Message: "Application request limit reached", Type: "OAuthException", Code: 4},
			rateLimit: true,
			temporary: true,
		},
		{
			name: "auth token expired",
			err:  MetaError{Message: "Error validating access token: Session has expired", Type: "OAuthException", Code: 190},
			auth: true,
		},
		{
			name:       "permission denied",
			err:        MetaError{Message: "Permissions error", Type: "OAuthException", Code: 10},
			permission: true,
		},
		{
			name:     "template code",
			err:      MetaError{Message: "Template does not exist", Type: "OAuthException", Code: 132001},
			template: true,
		},
		{
			name:      "temporary platform error",
			err:       MetaError{Message: "Please try again later", Type: "APIException", Code: 2},
			temporary: true,
		},
		{
			name:     "template subcode",
			err:      MetaError{Message: "Invalid parameter", Type: "OAuthException", Code: 100, ErrorSubcode: 2388024},
			template: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.IsRateLimit(); got != tt.rateLimit {
				t.Fatalf("IsRateLimit = %v, want %v", got, tt.rateLimit)
			}
			if got := tt.err.IsAuthError(); got != tt.auth {
				t.Fatalf("IsAuthError = %v, want %v", got, tt.auth)
			}
			if got := tt.err.IsPermissionError(); got != tt.permission {
				t.Fatalf("IsPermissionError = %v, want %v", got, tt.permission)
			}
			if got := tt.err.IsTemplateError(); got != tt.template {
				t.Fatalf("IsTemplateError = %v, want %v", got, tt.template)
			}
			if got := tt.err.IsTemporary(); got != tt.temporary {
				t.Fatalf("IsTemporary = %v, want %v", got, tt.temporary)
			}
		})
	}
}

func TestMetaErrorSafeMessageRedactsSecrets(t *testing.T) {
	err := MetaError{
		Message: `Error validating access_token=EAAGSECRET123456789 and app_secret: "topsecret" with bearer abc123`,
		Code:    190,
	}
	safe := err.SafeMessage()
	for _, forbidden := range []string{"EAAGSECRET123456789", "topsecret", "abc123"} {
		if strings.Contains(safe, forbidden) {
			t.Fatalf("SafeMessage leaked %q: %s", forbidden, safe)
		}
	}
	if !strings.Contains(safe, "[REDACTED]") {
		t.Fatalf("SafeMessage did not redact: %s", safe)
	}
	if strings.Contains(err.Error(), "EAAGSECRET123456789") {
		t.Fatalf("Error leaked token: %s", err.Error())
	}
}

func TestDecodeErrorCommonJSONShapes(t *testing.T) {
	body := `{"error":{"message":"Invalid OAuth access token.","type":"OAuthException","code":190,"fbtrace_id":"trace-123"}}`
	err := metaErrorFromBody(t, http.StatusUnauthorized, body)
	if err.MetaError == nil {
		t.Fatal("expected MetaError")
	}
	if err.MetaError.Code != 190 || err.MetaError.TraceID != "trace-123" {
		t.Fatalf("unexpected decoded error: %#v", err.MetaError)
	}
	if !err.MetaError.IsAuthError() {
		t.Fatalf("expected auth error")
	}
	if string(err.MetaError.RawBody) != body {
		t.Fatalf("raw body = %q", string(err.MetaError.RawBody))
	}
}

func TestDecodeErrorTemplateSubcodeShape(t *testing.T) {
	err := metaErrorFromBody(t, http.StatusBadRequest, `{"error":{"message":"Invalid parameter","type":"OAuthException","code":100,"error_subcode":2388024,"fbtrace_id":"trace-template"}}`)
	if err.MetaError == nil {
		t.Fatal("expected MetaError")
	}
	if !err.MetaError.IsTemplateError() {
		t.Fatalf("expected template error: %#v", err.MetaError)
	}
	if err.MetaError.ErrorSubcode != 2388024 {
		t.Fatalf("subcode = %d", err.MetaError.ErrorSubcode)
	}
}

func TestDecodeErrorFallbackForNonMetaBody(t *testing.T) {
	err := metaErrorFromBody(t, http.StatusInternalServerError, `not-json`)
	if err.MetaError != nil {
		t.Fatalf("expected nil MetaError, got %#v", err.MetaError)
	}
	if err.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d", err.StatusCode)
	}
}

func TestClientDecodedErrorHasRawBodyAndSafeError(t *testing.T) {
	body := `{"error":{"message":"Bad token access_token=EAAGSECRET123456789","type":"OAuthException","code":190,"fbtrace_id":"trace-123"}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	_, callErr := client.GetWABA(context.Background(), "EAAGSECRET123456789", "waba-123")
	if callErr == nil {
		t.Fatal("expected error")
	}
	var httpErr *HTTPError
	if !errors.As(callErr, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", callErr)
	}
	if string(httpErr.MetaError.RawBody) != body {
		t.Fatalf("raw body not preserved")
	}
	if strings.Contains(callErr.Error(), "EAAGSECRET123456789") {
		t.Fatalf("client error leaked token: %s", callErr.Error())
	}
}

func metaErrorFromBody(t *testing.T, status int, body string) *HTTPError {
	t.Helper()
	resp := &http.Response{
		StatusCode: status,
		Body:       ioNopCloser{strings.NewReader(body)},
	}
	err := decodeError(resp)
	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	return httpErr
}

type ioNopCloser struct {
	*strings.Reader
}

func (c ioNopCloser) Close() error {
	return nil
}
