package worker

import (
	"context"
	"doorman/internal/domain"
	natsinf "doorman/internal/infra/nats"
	"doorman/internal/pkg/correlation"
	"encoding/json"
	"fmt"
	"log"
)

// UserDeletedConsumer читает события caesar.user.deleted из NATS и сохраняет задачу в БД.
type UserDeletedConsumer struct {
	subscriber *natsinf.Subscriber
	tasks      taskScheduler
}

func NewUserDeletedConsumer(subscriber *natsinf.Subscriber, tasks taskScheduler) *UserDeletedConsumer {
	return &UserDeletedConsumer{subscriber: subscriber, tasks: tasks}
}

func (c *UserDeletedConsumer) Run(ctx context.Context) error {
	log.Printf("[user-deleted-consumer] подписка на %s", natsinf.SubjectUserDeleted)
	return c.subscriber.QueueSubscribe(ctx, natsinf.SubjectUserDeleted, natsinf.QueueUserDeleted, c.handle)
}

type userDeletedMsg struct {
	UserID        string `json:"user_id"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

func (c *UserDeletedConsumer) handle(ctx context.Context, data []byte) error {
	var msg userDeletedMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("разбор сообщения: %w", err)
	}

	if msg.CorrelationID != "" {
		ctx = correlation.WithID(ctx, msg.CorrelationID)
	}

	log.Printf("[user-deleted-consumer] correlation_id=%s user_id=%s",
		correlation.IDFromContext(ctx), msg.UserID)

	return c.tasks.Schedule(ctx, domain.UserDeleted, data)
}
