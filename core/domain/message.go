package domain

import (
	"time"

	"github.com/google/uuid"
)

type MessageStatus string

const (
	MessageStatusQueued    MessageStatus = "queued"
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
	MessageStatusFailed    MessageStatus = "failed"
)

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
	MediaType          *string // image, video, document, audio
	ProviderMessageID string  // Meta's wamid, Twilio SID, etc.
	Status            MessageStatus
	ErrorCode         *string
	ErrorMessage      *string
	Cost              *float64
	SentAt            *time.Time
	DeliveredAt       *time.Time
	ReadAt            *time.Time
	FailedAt          *time.Time
	CreatedAt         time.Time
	Metadata          map[string]string // custom tracking key-value pairs
}
