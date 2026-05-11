package sms

import (
	"testing"

	"github.com/ranakdinesh/spur-messaging/core/domain"
)

func TestTwilioStatusMappingUsesLifecycleStatuses(t *testing.T) {
	tests := map[string]domain.MessageStatus{
		"queued":    domain.MessageStatusQueued,
		"accepted":  domain.MessageStatusProviderSubmitted,
		"sending":   domain.MessageStatusProviderSubmitted,
		"sent":      domain.MessageStatusSent,
		"delivered": domain.MessageStatusDelivered,
		"failed":    domain.MessageStatusFailed,
	}

	for input, want := range tests {
		if got := mapTwilioStatus(input); got != want {
			t.Fatalf("expected %q to map to %q, got %q", input, want, got)
		}
	}
}
