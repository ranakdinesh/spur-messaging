package domain

import "testing"

func TestMessageStatusValidation(t *testing.T) {
	valid := []MessageStatus{
		MessageStatusCreated,
		MessageStatusValidated,
		MessageStatusQueued,
		MessageStatusProviderSubmitted,
		MessageStatusSent,
		MessageStatusDelivered,
		MessageStatusRead,
		MessageStatusOpened,
		MessageStatusClicked,
		MessageStatusReplied,
		MessageStatusFailed,
		MessageStatusCancelled,
		MessageStatusExpired,
		MessageStatusSuppressed,
	}

	for _, status := range valid {
		if !IsValidMessageStatus(status) {
			t.Fatalf("expected %q to be valid", status)
		}
		if MessageStatusRank(status) < 0 {
			t.Fatalf("expected %q to have a non-negative rank", status)
		}
	}

	if IsValidMessageStatus("unknown") {
		t.Fatal("expected unknown status to be invalid")
	}
	if MessageStatusRank("unknown") != -1 {
		t.Fatal("expected unknown status rank to be -1")
	}
}

func TestMessageStatusRankProgression(t *testing.T) {
	ordered := []MessageStatus{
		MessageStatusCreated,
		MessageStatusValidated,
		MessageStatusQueued,
		MessageStatusProviderSubmitted,
		MessageStatusSent,
		MessageStatusDelivered,
		MessageStatusOpened,
		MessageStatusClicked,
		MessageStatusReplied,
		MessageStatusFailed,
	}

	for i := 1; i < len(ordered); i++ {
		prev := ordered[i-1]
		next := ordered[i]
		if MessageStatusRank(next) <= MessageStatusRank(prev) {
			t.Fatalf("expected %q to rank after %q", next, prev)
		}
	}

	if MessageStatusRank(MessageStatusRead) != MessageStatusRank(MessageStatusOpened) {
		t.Fatal("expected WhatsApp read and email opened to have the same delivery-depth rank")
	}
}
