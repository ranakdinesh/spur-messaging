package email

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

func TestDevEmailProviderWritesOutboxRecord(t *testing.T) {
	outbox := t.TempDir()
	t.Setenv("MESSAGING_DEV_EMAIL_OUTBOX", outbox)

	provider := NewDevEmailProvider()
	result, err := provider.SendEmail(context.Background(), &domain.ProviderConfig{}, ports.EmailSendRequest{
		To:        "new.user@example.com",
		FromEmail: "noreply@citual.test",
		Subject:   "Verify your email",
		HTMLBody:  `<a href="http://localhost:8080/auth/email/verify?token=test">Verify</a>`,
		TextBody:  "Verify: http://localhost:8080/auth/email/verify?token=test",
		Metadata:  map[string]string{"kind": "email_verification"},
	})
	if err != nil {
		t.Fatalf("SendEmail returned error: %v", err)
	}
	if result.Status != domain.MessageStatusSent {
		t.Fatalf("expected sent status, got %s", result.Status)
	}

	files, err := os.ReadDir(outbox)
	if err != nil {
		t.Fatalf("read outbox: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected one outbox file, got %d", len(files))
	}
	raw, err := os.ReadFile(filepath.Join(outbox, files[0].Name()))
	if err != nil {
		t.Fatalf("read outbox file: %v", err)
	}
	var record devEmailRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		t.Fatalf("unmarshal outbox record: %v", err)
	}
	if record.To != "new.user@example.com" || record.Subject != "Verify your email" || record.Metadata["kind"] != "email_verification" {
		t.Fatalf("unexpected outbox record: %#v", record)
	}
}
