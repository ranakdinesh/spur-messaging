package ports

import (
	"context"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
)

type MessageQueue interface {
	Enqueue(ctx context.Context, msg QueueMessage) error
	EnqueueBulk(ctx context.Context, msgs []QueueMessage) error
	// Dequeue is called by the worker — implemented as Redis Streams consumer
	StartConsumer(ctx context.Context, handler func(ctx context.Context, msg QueueMessage) error) error
	Stop()
}

type QueueMessage struct {
	MessageID uuid.UUID      `json:"message_id"`
	TenantID  uuid.UUID      `json:"tenant_id"`
	Channel   domain.Channel `json:"channel"`
	Priority  int            `json:"priority"` // 0 = normal, 1 = high (OTP/auth)
}
