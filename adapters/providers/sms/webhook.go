package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

// ParseMSG91Webhook for MSG91
func ParseMSG91Webhook(ctx context.Context, cfg *domain.ProviderConfig, headers http.Header, body []byte) ([]ports.WebhookEvent, error) {
	// MSG91 delivery receipt format varies, but usually it's a JSON array or object
	// Documentation: https://msg91.com/help/msg91-delivery-report-webhook

	var data []struct {
		RequestID string `json:"requestId"`
		Status    string `json:"status"`
		Mobile    string `json:"mobile"`
		Desc      string `json:"desc"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		// Try single object if array fails
		var single struct {
			RequestID string `json:"requestId"`
			Status    string `json:"status"`
			Mobile    string `json:"mobile"`
			Desc      string `json:"desc"`
		}
		if err := json.Unmarshal(body, &single); err != nil {
			return nil, fmt.Errorf("failed to decode msg91 webhook: %w", err)
		}
		data = append(data, single)
	}

	events := make([]ports.WebhookEvent, 0, len(data))
	for _, item := range data {
		status := mapMsg91Status(item.Status)
		events = append(events, ports.WebhookEvent{
			Type:              ports.WebhookEventStatusUpdate,
			ProviderMessageID: item.RequestID,
			Status:            &status,
			Timestamp:         time.Now(),
		})
	}

	return events, nil
}

// ParseTwilioWebhook for Twilio
func ParseTwilioWebhook(ctx context.Context, cfg *domain.ProviderConfig, headers http.Header, body []byte) ([]ports.WebhookEvent, error) {
	// Twilio sends webhooks as application/x-www-form-urlencoded
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse twilio webhook body: %w", err)
	}

	msgSID := values.Get("MessageSid")
	twilioStatus := values.Get("MessageStatus")

	if msgSID == "" {
		return nil, fmt.Errorf("missing MessageSid in twilio webhook")
	}

	status := mapTwilioStatus(twilioStatus)

	return []ports.WebhookEvent{
		{
			Type:              ports.WebhookEventStatusUpdate,
			ProviderMessageID: msgSID,
			Status:            &status,
			Timestamp:         time.Now(),
		},
	}, nil
}

func mapMsg91Status(s string) domain.MessageStatus {
	switch s {
	case "1": // Delivered
		return domain.MessageStatusDelivered
	case "2": // Failed
		return domain.MessageStatusFailed
	case "16": // Rejected
		return domain.MessageStatusFailed
	default:
		return domain.MessageStatusSent
	}
}

func mapTwilioStatus(s string) domain.MessageStatus {
	switch s {
	case "queued":
		return domain.MessageStatusQueued
	case "accepted", "sending":
		return domain.MessageStatusProviderSubmitted
	case "sent":
		return domain.MessageStatusSent
	case "delivered":
		return domain.MessageStatusDelivered
	case "undelivered", "failed":
		return domain.MessageStatusFailed
	case "read":
		return domain.MessageStatusRead
	default:
		return domain.MessageStatusSent
	}
}
