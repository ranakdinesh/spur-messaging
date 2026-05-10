package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
	"github.com/redis/go-redis/v9"
)

const (
	StreamOutbox         = "messaging:outbox"
	StreamOutboxPriority = "messaging:outbox:priority"
	ConsumerGroup        = "spur-messaging-workers"
)

type RedisQueue struct {
	client *redis.Client
	wg     sync.WaitGroup
	done   chan struct{}
	shared bool // true = client owned externally, don't close on Stop()
}

func NewRedisQueue(redisURL string) (*RedisQueue, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opts)
	return &RedisQueue{
		client: client,
		done:   make(chan struct{}),
	}, nil
}

// NewRedisQueueFromClient creates a RedisQueue from an existing Redis client.
// Used in production where spur-template provides the shared Redis connection.
// The caller owns the client lifecycle — Stop() will NOT close it.
func NewRedisQueueFromClient(client *redis.Client) *RedisQueue {
	return &RedisQueue{
		client: client,
		done:   make(chan struct{}),
		shared: true,
	}
}

func (q *RedisQueue) Enqueue(ctx context.Context, msg ports.QueueMessage) error {
	stream := StreamOutbox
	if msg.Priority > 0 {
		stream = StreamOutboxPriority
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	err = q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]interface{}{"payload": data},
	}).Err()

	if err != nil {
		// Section 10A.3: Enqueue failure (Redis down)
		return domain.ErrQueueUnavailable
	}
	return nil
}

func (q *RedisQueue) EnqueueBulk(ctx context.Context, msgs []ports.QueueMessage) error {
	for _, msg := range msgs {
		if err := q.Enqueue(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (q *RedisQueue) StartConsumer(ctx context.Context, handler func(ctx context.Context, msg ports.QueueMessage) error) error {
	// Ensure consumer groups exist
	for _, stream := range []string{StreamOutbox, StreamOutboxPriority} {
		err := q.client.XGroupCreateMkStream(ctx, stream, ConsumerGroup, "0").Err()
		if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
			return err
		}
	}

	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		consumerName := fmt.Sprintf("worker-%d", time.Now().UnixNano())

		for {
			select {
			case <-q.done:
				return
			case <-ctx.Done():
				return
			default:
				// 1. Try Priority Stream
				processed, err := q.consume(ctx, StreamOutboxPriority, consumerName, handler)
				if err != nil {
					// Log error? For now just continue
					time.Sleep(time.Second)
					continue
				}
				if processed {
					continue // Check priority again if we processed something
				}

				// 2. Try Normal Stream
				processed, err = q.consume(ctx, StreamOutbox, consumerName, handler)
				if err != nil {
					time.Sleep(time.Second)
					continue
				}
				if !processed {
					time.Sleep(time.Second) // Idle wait
				}
			}
		}
	}()

	return nil
}

func (q *RedisQueue) consume(ctx context.Context, stream, consumer string, handler func(ctx context.Context, msg ports.QueueMessage) error) (bool, error) {
	// Read from group
	streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    ConsumerGroup,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count:    1,
		Block:    time.Millisecond * 100,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	for _, s := range streams {
		for _, m := range s.Messages {
			payload, ok := m.Values["payload"].(string)
			if !ok {
				continue
			}

			var msg ports.QueueMessage
			if err := json.Unmarshal([]byte(payload), &msg); err != nil {
				continue
			}

			// Handle message
			if err := handler(ctx, msg); err != nil {
				// Section 10A.3: Only ACK after successful send + DB update.
				// If handler returns error, we don't ACK, it stays in PEL.
				return false, err
			}

			// ACK message
			return true, q.client.XAck(ctx, stream, ConsumerGroup, m.ID).Err()
		}
	}

	return false, nil
}

func (q *RedisQueue) Stop() {
	close(q.done)
	q.wg.Wait()
	if !q.shared {
		q.client.Close()
	}
}
