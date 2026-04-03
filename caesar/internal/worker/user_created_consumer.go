package worker

import (
	"caesar/internal/domain"
	natsinf "caesar/internal/infra/nats"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

type UserCreatedConsumer struct {
	subscriber *natsinf.Subscriber
	tasks      taskScheduler
}

func NewUserCreatedConsumer(subscriber *natsinf.Subscriber, tasks taskScheduler) *UserCreatedConsumer {
	return &UserCreatedConsumer{subscriber: subscriber, tasks: tasks}
}

func (c *UserCreatedConsumer) Run(ctx context.Context) error {
	slog.Info("user-created-consumer: подписка", "subject", natsinf.SubjectUserCreated)
	return c.subscriber.QueueSubscribe(ctx, natsinf.SubjectUserCreated, natsinf.QueueUserCreated, c.handle)
}

type userCreatedMsg struct {
	UserID  string `json:"user_id"`
	Phone   string `json:"phone"`
	Name    string `json:"name"`
	Surname string `json:"surname"`
}

func (c *UserCreatedConsumer) handle(ctx context.Context, data []byte) error {
	var msg userCreatedMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}
	slog.InfoContext(ctx, "user-created-consumer: получено", "user_id", msg.UserID)
	return c.tasks.Schedule(ctx, domain.UserCreated, data)
}
