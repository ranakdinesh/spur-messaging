package domain

import (
	"time"

	"github.com/google/uuid"
)

type ConversationStatus string

const (
	ConversationStatusOpen     ConversationStatus = "open"
	ConversationStatusPending  ConversationStatus = "pending"
	ConversationStatusResolved ConversationStatus = "resolved"
	ConversationStatusClosed   ConversationStatus = "closed"
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
	AssignedAgentID    *uuid.UUID
	AssignedTeam       *string
	Priority           ConversationPriority
	Tags               []string
	Notes              []ConversationNote
	LastInboundAt      *time.Time
	LastOutboundAt     *time.Time
	ServiceWindowUntil *time.Time
	FirstResponseDueAt *time.Time
	ResolutionDueAt    *time.Time
	ClosedAt           *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type ConversationPriority string

const (
	ConversationPriorityLow    ConversationPriority = "low"
	ConversationPriorityMedium ConversationPriority = "medium"
	ConversationPriorityHigh   ConversationPriority = "high"
	ConversationPriorityUrgent ConversationPriority = "urgent"
)

type ConversationNote struct {
	ID        uuid.UUID `json:"id"`
	AuthorID  uuid.UUID `json:"author_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

func IsValidConversationStatus(status ConversationStatus) bool {
	switch status {
	case ConversationStatusOpen, ConversationStatusPending, ConversationStatusResolved, ConversationStatusClosed:
		return true
	default:
		return false
	}
}

func IsValidConversationHandoffStatus(status ConversationHandoffStatus) bool {
	switch status {
	case ConversationHandoffBot, ConversationHandoffAgent, ConversationHandoffClosed, ConversationHandoffWaiting:
		return true
	default:
		return false
	}
}

func IsValidConversationPriority(priority ConversationPriority) bool {
	switch priority {
	case ConversationPriorityLow, ConversationPriorityMedium, ConversationPriorityHigh, ConversationPriorityUrgent:
		return true
	default:
		return false
	}
}
