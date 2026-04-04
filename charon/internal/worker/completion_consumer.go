package worker

import (
	"charon/internal/domain"
	natsinf "charon/internal/infra/nats"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

type taskScheduler interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
}

type CompletionConsumer struct {
	subscriber *natsinf.Subscriber
	tasks      taskScheduler
}

func NewCompletionConsumer(subscriber *natsinf.Subscriber, tasks taskScheduler) *CompletionConsumer {
	return &CompletionConsumer{subscriber: subscriber, tasks: tasks}
}

func (c *CompletionConsumer) Run(ctx context.Context) error {
	slog.Info("completion-consumer: subscribing", "subject", natsinf.SubjectCompletionRequested)
	return c.subscriber.QueueSubscribe(ctx, natsinf.SubjectCompletionRequested, natsinf.QueueCompletionRequested, c.handle)
}

func (c *CompletionConsumer) handle(ctx context.Context, data []byte) error {
	var msg domain.CompletionRequest
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}
	slog.InfoContext(ctx, "completion-consumer: received",
		"request_id", msg.RequestID, "service", msg.Service, "model", msg.Model)
	return c.tasks.Schedule(ctx, domain.CompletionRequested, data)
}
