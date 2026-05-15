package email

import (
	"strings"
	"testing"

	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

func TestBuildSMTPMessageIncludesSafeHeadersAndBodies(t *testing.T) {
	msg, err := buildSMTPMessage(ports.EmailSendRequest{
		To:        "user@example.com",
		FromEmail: "noreply@example.com",
		FromName:  "Citual",
		ReplyTo:   "support@example.com",
		Subject:   "Verify your email",
		HTMLBody:  "<p>Verify</p>",
		TextBody:  "Verify",
		Headers: map[string]string{
			"List-Unsubscribe": "<https://example.com/unsubscribe>",
			"Bad\r\nHeader":    "should-not-appear",
		},
		Metadata: map[string]string{
			"X-Correlation-ID": "registration-123",
			"subject":          "not a header",
		},
	}, "<test@example.com>")
	if err != nil {
		t.Fatalf("buildSMTPMessage returned error: %v", err)
	}
	raw := string(msg)
	for _, want := range []string{
		`From: "Citual" <noreply@example.com>`,
		"To: user@example.com",
		"Reply-To: support@example.com",
		"List-Unsubscribe: <https://example.com/unsubscribe>",
		"X-Correlation-Id: registration-123",
		"Content-Type: multipart/alternative;",
		"Verify",
		"<p>Verify</p>",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("expected SMTP message to contain %q, got:\n%s", want, raw)
		}
	}
	if strings.Contains(raw, "should-not-appear") {
		t.Fatalf("unsafe header value leaked into SMTP message:\n%s", raw)
	}
}

func TestSMTPCredentialsUseEnvironmentFallback(t *testing.T) {
	t.Setenv("MESSAGING_SMTP_HOST", "smtp.example.com")
	t.Setenv("MESSAGING_SMTP_PORT", "2525")
	t.Setenv("MESSAGING_SMTP_USERNAME", "user")
	t.Setenv("MESSAGING_SMTP_PASSWORD", "password")
	t.Setenv("MESSAGING_SMTP_AUTH", "plain")
	t.Setenv("MESSAGING_SMTP_TLS_MODE", "starttls")

	creds, err := smtpCredentials(&domain.ProviderConfig{})
	if err != nil {
		t.Fatalf("smtpCredentials returned error: %v", err)
	}
	if creds.SMTPHost != "smtp.example.com" || creds.SMTPPort != 2525 || creds.SMTPUsername != "user" || creds.SMTPPassword != "password" || creds.SMTPAuth != "plain" || creds.SMTPTLSMode != "starttls" {
		t.Fatalf("unexpected SMTP credentials from environment: %#v", creds)
	}
}
