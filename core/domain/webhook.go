package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"time"

	"github.com/google/uuid"
)

type WebhookEventType string

const (
	WebhookEventMessageCreated   WebhookEventType = "message.created"
	WebhookEventMessageSent      WebhookEventType = "message.sent"
	WebhookEventMessageDelivered WebhookEventType = "message.delivered"
	WebhookEventMessageRead      WebhookEventType = "message.read"
	WebhookEventMessageOpened    WebhookEventType = "message.opened"
	WebhookEventMessageClicked   WebhookEventType = "message.clicked"
	WebhookEventMessageFailed    WebhookEventType = "message.failed"
	WebhookEventMessageReplied   WebhookEventType = "message.replied"
	WebhookEventTemplateApproved WebhookEventType = "template.approved"
	WebhookEventTemplateRejected WebhookEventType = "template.rejected"
	WebhookEventContactOptedIn   WebhookEventType = "contact.opted_in"
	WebhookEventContactOptedOut  WebhookEventType = "contact.opted_out"
	WebhookEventCampaignStarted  WebhookEventType = "campaign.started"
	WebhookEventCampaignComplete WebhookEventType = "campaign.completed"
	WebhookEventCampaignFailed   WebhookEventType = "campaign.failed"
	WebhookEventWalletLowBalance WebhookEventType = "wallet.low_balance"
	WebhookEventTest             WebhookEventType = "webhook.test"
)

var validWebhookEvents = map[WebhookEventType]struct{}{
	WebhookEventMessageCreated:   {},
	WebhookEventMessageSent:      {},
	WebhookEventMessageDelivered: {},
	WebhookEventMessageRead:      {},
	WebhookEventMessageOpened:    {},
	WebhookEventMessageClicked:   {},
	WebhookEventMessageFailed:    {},
	WebhookEventMessageReplied:   {},
	WebhookEventTemplateApproved: {},
	WebhookEventTemplateRejected: {},
	WebhookEventContactOptedIn:   {},
	WebhookEventContactOptedOut:  {},
	WebhookEventCampaignStarted:  {},
	WebhookEventCampaignComplete: {},
	WebhookEventCampaignFailed:   {},
	WebhookEventWalletLowBalance: {},
	WebhookEventTest:             {},
}

func IsValidWebhookEvent(event WebhookEventType) bool {
	_, ok := validWebhookEvents[event]
	return ok
}

func IsValidWebhookURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	return err == nil && u.Scheme == "https" && u.Host != ""
}

type WebhookEndpoint struct {
	ID           uuid.UUID          `json:"id"`
	TenantID     uuid.UUID          `json:"tenant_id"`
	URL          string             `json:"url"`
	Secret       string             `json:"secret,omitempty"`
	Events       []WebhookEventType `json:"events"`
	IsActive     bool               `json:"is_active"`
	FailureCount int                `json:"failure_count"`
	DisabledAt   *time.Time         `json:"disabled_at,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

func (w WebhookEndpoint) SubscribesTo(event WebhookEventType) bool {
	return w.IsActive && slices.Contains(w.Events, event)
}

type WebhookDeliveryStatus string

const (
	WebhookDeliveryPending   WebhookDeliveryStatus = "pending"
	WebhookDeliverySucceeded WebhookDeliveryStatus = "succeeded"
	WebhookDeliveryRetrying  WebhookDeliveryStatus = "retrying"
	WebhookDeliveryFailed    WebhookDeliveryStatus = "failed"
)

type WebhookDelivery struct {
	ID             uuid.UUID             `json:"id"`
	TenantID       uuid.UUID             `json:"tenant_id"`
	WebhookID      uuid.UUID             `json:"webhook_id"`
	EventID        uuid.UUID             `json:"event_id"`
	EventType      WebhookEventType      `json:"event_type"`
	Payload        json.RawMessage       `json:"payload"`
	Status         WebhookDeliveryStatus `json:"status"`
	AttemptCount   int                   `json:"attempt_count"`
	NextAttemptAt  *time.Time            `json:"next_attempt_at,omitempty"`
	LastAttemptAt  *time.Time            `json:"last_attempt_at,omitempty"`
	ResponseStatus *int                  `json:"response_status,omitempty"`
	ResponseBody   *string               `json:"response_body,omitempty"`
	ErrorMessage   *string               `json:"error_message,omitempty"`
	Signature      string                `json:"signature,omitempty"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

func SignWebhookPayload(secret string, timestamp time.Time, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(fmt.Sprintf("%d.", timestamp.Unix())))
	_, _ = mac.Write(payload)
	return "v1=" + hex.EncodeToString(mac.Sum(nil))
}
