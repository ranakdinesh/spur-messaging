package domain

import (
	"time"

	"github.com/google/uuid"
)

type MessageStatus string

const (
	MessageStatusCreated           MessageStatus = "created"
	MessageStatusValidated         MessageStatus = "validated"
	MessageStatusQueued            MessageStatus = "queued"
	MessageStatusProviderSubmitted MessageStatus = "provider_submitted"
	MessageStatusSent              MessageStatus = "sent"
	MessageStatusDelivered         MessageStatus = "delivered"
	MessageStatusRead              MessageStatus = "read"
	MessageStatusOpened            MessageStatus = "opened"
	MessageStatusClicked           MessageStatus = "clicked"
	MessageStatusReplied           MessageStatus = "replied"
	MessageStatusFailed            MessageStatus = "failed"
	MessageStatusCancelled         MessageStatus = "cancelled"
	MessageStatusExpired           MessageStatus = "expired"
	MessageStatusSuppressed        MessageStatus = "suppressed"
)

var messageStatusRanks = map[MessageStatus]int{
	MessageStatusCreated:           0,
	MessageStatusValidated:         10,
	MessageStatusQueued:            20,
	MessageStatusProviderSubmitted: 30,
	MessageStatusSent:              40,
	MessageStatusDelivered:         50,
	MessageStatusRead:              60,
	MessageStatusOpened:            60,
	MessageStatusClicked:           70,
	MessageStatusReplied:           80,
	MessageStatusFailed:            100,
	MessageStatusCancelled:         100,
	MessageStatusExpired:           100,
	MessageStatusSuppressed:        100,
}

func IsValidMessageStatus(status MessageStatus) bool {
	_, ok := messageStatusRanks[status]
	return ok
}

func MessageStatusRank(status MessageStatus) int {
	rank, ok := messageStatusRanks[status]
	if !ok {
		return -1
	}
	return rank
}

type MessageType string

const (
	MessageTypeTemplate    MessageType = "template"
	MessageTypeText        MessageType = "text"
	MessageTypeMedia       MessageType = "media"
	MessageTypeInteractive MessageType = "interactive"
	MessageTypeLocation    MessageType = "location"
)

type Message struct {
	ID                uuid.UUID
	TenantID          uuid.UUID
	CampaignID        *uuid.UUID
	ConversationID    *uuid.UUID
	Channel           Channel
	Direction         string // "outbound" or "inbound"
	Recipient         string // E.164 phone or email
	Sender            string // platform phone number or email
	MessageType       MessageType
	TemplateID        *uuid.UUID
	TemplateName      *string
	TemplateParams    map[string]string
	TextBody          *string
	MediaURL          *string
	MediaType         *string // image, video, document, audio
	ProviderMessageID string  // Meta's wamid, Twilio SID, etc.
	IdempotencyKey    *string
	Status            MessageStatus
	ErrorCode         *string
	ErrorMessage      *string
	Cost              *float64
	SentAt            *time.Time
	DeliveredAt       *time.Time
	ReadAt            *time.Time
	FailedAt          *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
	Metadata          map[string]string // custom tracking key-value pairs
}
