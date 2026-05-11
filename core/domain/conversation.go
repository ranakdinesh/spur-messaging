package domain

import (
	"time"

	"github.com/google/uuid"
)

type ConversationStatus string

const (
	ConversationStatusOpen   ConversationStatus = "open"
	ConversationStatusClosed ConversationStatus = "closed"
)

type ConversationHandoffStatus string

const (
	ConversationHandoffBot     ConversationHandoffStatus = "bot"
	ConversationHandoffAgent   ConversationHandoffStatus = "agent"
	ConversationHandoffClosed  ConversationHandoffStatus = "closed"
	ConversationHandoffWaiting ConversationHandoffStatus = "waiting_customer"
)

type Conversation struct {
	ID                 uuid.UUID
	TenantID           uuid.UUID
	Channel            Channel
	Recipient          string
	Status             ConversationStatus
	HandoffStatus      ConversationHandoffStatus
	LastInboundAt      *time.Time
	LastOutboundAt     *time.Time
	ServiceWindowUntil *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
