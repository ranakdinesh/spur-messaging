package email

import (
	"testing"

	"github.com/ranakdinesh/spur-messaging/core/domain"
)

func TestEmailWebhookStatusMappingUsesLifecycleStatuses(t *testing.T) {
	tests := []struct {
		name string
		got  *domain.MessageStatus
		want domain.MessageStatus
	}{
		{name: "sendgrid processed", got: mapSendGridStatus("processed"), want: domain.MessageStatusProviderSubmitted},
		{name: "sendgrid open", got: mapSendGridStatus("open"), want: domain.MessageStatusOpened},
		{name: "sendgrid click", got: mapSendGridStatus("click"), want: domain.MessageStatusClicked},
		{name: "mailgun opened", got: mapMailgunStatus("opened"), want: domain.MessageStatusOpened},
		{name: "mailgun clicked", got: mapMailgunStatus("clicked"), want: domain.MessageStatusClicked},
		{name: "postmark open", got: mapPostmarkStatus("Open"), want: domain.MessageStatusOpened},
		{name: "postmark click", got: mapPostmarkStatus("Click"), want: domain.MessageStatusClicked},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got == nil {
				t.Fatalf("expected %q, got nil", tt.want)
			}
			if *tt.got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, *tt.got)
			}
		})
	}
}
