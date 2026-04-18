package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"riddler/internal/domain"
	natsinf "riddler/internal/infra/nats"
)

type gradingCompletedScheduler interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
}

type GradingCompletedConsumer struct {
	subscriber *natsinf.Subscriber
	tasks      gradingCompletedScheduler
}

func NewGradingCompletedConsumer(subscriber *natsinf.Subscriber, tasks gradingCompletedScheduler) *GradingCompletedConsumer {
	return &GradingCompletedConsumer{subscriber: subscriber, tasks: tasks}
}

func (c *GradingCompletedConsumer) Run(ctx context.Context) error {
	slog.Info("grading-completed-consumer: подписка", "subject", natsinf.SubjectQuizGradeCompleted)
	return c.subscriber.QueueSubscribe(ctx, natsinf.SubjectQuizGradeCompleted, "riddler-grading-completed", c.handle)
}

func (c *GradingCompletedConsumer) handle(ctx context.Context, data []byte) error {
	var msg struct {
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}
	slog.InfoContext(ctx, "grading-completed-consumer: получено", "request_id", msg.RequestID)
	return c.tasks.Schedule(ctx, domain.TaskTypeGradingCompleted, data)
}
