package email

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

// ParseSendGridWebhook parses SendGrid v3 event webhooks.
func ParseSendGridWebhook(ctx context.Context, cfg *domain.ProviderConfig, headers http.Header, body []byte) ([]ports.WebhookEvent, error) {
	var events []map[string]any
	if err := json.Unmarshal(body, &events); err != nil {
		return nil, fmt.Errorf("unmarshal sendgrid events: %w", err)
	}

	webhookEvents := make([]ports.WebhookEvent, 0, len(events))
	for _, event := range events {
		eventType, _ := event["event"].(string)
		msgID, _ := event["message_id"].(string)
		timestampRaw, _ := event["timestamp"].(float64)

		status := mapSendGridStatus(eventType)
		if status == nil {
			continue
		}

		webhookEvents = append(webhookEvents, ports.WebhookEvent{
			Type:              ports.WebhookEventStatusUpdate,
			ProviderMessageID: msgID,
			Status:            status,
			Timestamp:         time.Unix(int64(timestampRaw), 0),
		})
	}

	return webhookEvents, nil
}

// ParseMailgunWebhook parses Mailgun event webhooks.
func ParseMailgunWebhook(ctx context.Context, cfg *domain.ProviderConfig, headers http.Header, body []byte) ([]ports.WebhookEvent, error) {
	var payload struct {
		EventData struct {
			Event     string  `json:"event"`
			ID        string  `json:"id"`
			Timestamp float64 `json:"timestamp"`
			Message   struct {
				Headers struct {
					MessageID string `json:"message-id"`
				} `json:"headers"`
			} `json:"message"`
			UserVariables map[string]string `json:"user-variables"`
		} `json:"event-data"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal mailgun event: %w", err)
	}

	status := mapMailgunStatus(payload.EventData.Event)
	if status == nil {
		return nil, nil
	}

	msgID := payload.EventData.Message.Headers.MessageID
	if v, ok := payload.EventData.UserVariables["message_id"]; ok {
		msgID = v
	}

	return []ports.WebhookEvent{{
		Type:              ports.WebhookEventStatusUpdate,
		ProviderMessageID: msgID,
		Status:            status,
		Timestamp:         time.Unix(int64(payload.EventData.Timestamp), 0),
	}}, nil
}

// ParsePostmarkWebhook parses Postmark event webhooks.
func ParsePostmarkWebhook(ctx context.Context, cfg *domain.ProviderConfig, headers http.Header, body []byte) ([]ports.WebhookEvent, error) {
	var event map[string]any
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("unmarshal postmark event: %w", err)
	}

	recordType, _ := event["RecordType"].(string)
	msgID, _ := event["MessageID"].(string)
	receivedAt, _ := event["ReceivedAt"].(string)

	status := mapPostmarkStatus(recordType)
	if status == nil {
		return nil, nil
	}

	ts, _ := time.Parse(time.RFC3339, receivedAt)
	if ts.IsZero() {
		ts = time.Now()
	}

	return []ports.WebhookEvent{{
		Type:              ports.WebhookEventStatusUpdate,
		ProviderMessageID: msgID,
		Status:            status,
		Timestamp:         ts,
	}}, nil
}

func mapSendGridStatus(event string) *domain.MessageStatus {
	var s domain.MessageStatus
	switch event {
	case "delivered":
		s = domain.MessageStatusDelivered
	case "bounce", "dropped":
		s = domain.MessageStatusFailed
	case "open":
		s = domain.MessageStatusRead // We map open to read for analytics
	default:
		return nil
	}
	return &s
}

func mapMailgunStatus(event string) *domain.MessageStatus {
	var s domain.MessageStatus
	switch event {
	case "delivered":
		s = domain.MessageStatusDelivered
	case "failed":
		s = domain.MessageStatusFailed
	case "opened":
		s = domain.MessageStatusRead
	default:
		return nil
	}
	return &s
}

func mapPostmarkStatus(recordType string) *domain.MessageStatus {
	var s domain.MessageStatus
	switch recordType {
	case "Delivery":
		s = domain.MessageStatusDelivered
	case "Bounce":
		s = domain.MessageStatusFailed
	case "Open":
		s = domain.MessageStatusRead
	default:
		return nil
	}
	return &s
}
