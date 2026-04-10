package worker

import (
	"charon/internal/domain"
	natsinf "charon/internal/infra/nats"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel"
)

type GradingConsumer struct {
	subscriber *natsinf.Subscriber
	tasks      taskScheduler
}

func NewGradingConsumer(subscriber *natsinf.Subscriber, tasks taskScheduler) *GradingConsumer {
	return &GradingConsumer{subscriber: subscriber, tasks: tasks}
}

func (c *GradingConsumer) Run(ctx context.Context) error {
	slog.Info("grading-consumer: subscribing", "subject", natsinf.SubjectQuizGradeRequested)
	return c.subscriber.QueueSubscribe(ctx, natsinf.SubjectQuizGradeRequested, natsinf.QueueQuizGradeRequested, c.handle)
}

func (c *GradingConsumer) handle(ctx context.Context, data []byte) error {
	ctx, span := otel.Tracer("charon").Start(ctx, "worker.grading_consumer")
	defer span.End()

	var msg domain.QuizGradeRequest
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}
	slog.InfoContext(ctx, "grading-consumer: received",
		"request_id", msg.RequestID, "answers", len(msg.Answers))
	return c.tasks.Schedule(ctx, domain.QuizGradeRequested, data)
}
