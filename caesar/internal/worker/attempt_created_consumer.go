package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"caesar/internal/domain"
	natsinf "caesar/internal/infra/nats"
)

type AttemptCreatedConsumer struct {
	subscriber *natsinf.Subscriber
	tasks      taskScheduler
}

func NewAttemptCreatedConsumer(subscriber *natsinf.Subscriber, tasks taskScheduler) *AttemptCreatedConsumer {
	return &AttemptCreatedConsumer{subscriber: subscriber, tasks: tasks}
}

func (c *AttemptCreatedConsumer) Run(ctx context.Context) error {
	slog.Info("attempt-created-consumer: подписка", "subject", natsinf.SubjectAttemptCreated)
	return c.subscriber.QueueSubscribe(ctx, natsinf.SubjectAttemptCreated, natsinf.QueueAttemptCreated, c.handle)
}

type attemptCreatedMsg struct {
	AttemptID string `json:"attempt_id"`
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
}

func (c *AttemptCreatedConsumer) handle(ctx context.Context, data []byte) error {
	var msg attemptCreatedMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}
	slog.InfoContext(ctx, "attempt-created-consumer: получено", "attempt_id", msg.AttemptID)
	return c.tasks.Schedule(ctx, domain.AttemptCreated, data)
}
