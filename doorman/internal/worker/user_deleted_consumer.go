package worker

import (
	"context"
	"doorman/internal/domain"
	natsinf "doorman/internal/infra/nats"
	"encoding/json"
	"fmt"
	"log/slog"
)

type UserDeletedConsumer struct {
	subscriber *natsinf.Subscriber
	tasks      taskScheduler
}

func NewUserDeletedConsumer(subscriber *natsinf.Subscriber, tasks taskScheduler) *UserDeletedConsumer {
	return &UserDeletedConsumer{subscriber: subscriber, tasks: tasks}
}

func (c *UserDeletedConsumer) Run(ctx context.Context) error {
	slog.Info("user-deleted-consumer: подписка", "subject", natsinf.SubjectUserDeleted)
	return c.subscriber.QueueSubscribe(ctx, natsinf.SubjectUserDeleted, natsinf.QueueUserDeleted, c.handle)
}

type userDeletedMsg struct {
	UserID string `json:"user_id"`
}

func (c *UserDeletedConsumer) handle(ctx context.Context, data []byte) error {
	var msg userDeletedMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}
	slog.InfoContext(ctx, "user-deleted-consumer: получено", "user_id", msg.UserID)
	return c.tasks.Schedule(ctx, domain.UserDeleted, data)
}
