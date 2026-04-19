package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"riddler/internal/domain"
	natsinf "riddler/internal/infra/nats"
	"riddler/internal/infra/telemetry"
)

type generationCompletedScheduler interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
}

const generationCompletedConsumer = "riddler-generation-completed"

type GenerationCompletedConsumer struct {
	subscriber *natsinf.JetStreamSubscriber
	tasks      generationCompletedScheduler
}

func NewGenerationCompletedConsumer(subscriber *natsinf.JetStreamSubscriber, tasks generationCompletedScheduler) *GenerationCompletedConsumer {
	return &GenerationCompletedConsumer{subscriber: subscriber, tasks: tasks}
}

func (c *GenerationCompletedConsumer) Run(ctx context.Context) error {
	slog.Info("generation-completed-consumer: подписка", "subject", natsinf.SubjectGenerationCompleted)
	return c.subscriber.Subscribe(
		ctx,
		natsinf.SubjectGenerationCompleted,
		natsinf.StreamGenerationCompleted,
		generationCompletedConsumer,
		c.handle,
	)
}

type generationCompletedMsg struct {
	JobID    uuid.UUID `json:"job_id"`
	QuizID   uuid.UUID `json:"quiz_id"`
	TraceCtx string    `json:"trace_ctx"`
}

func (c *GenerationCompletedConsumer) handle(ctx context.Context, data []byte) error {
	var msg generationCompletedMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}
	ctx = telemetry.Extract(ctx, msg.TraceCtx)
	slog.InfoContext(ctx, "generation-completed-consumer: получено", "job_id", msg.JobID, "quiz_id", msg.QuizID)
	return c.tasks.Schedule(ctx, domain.TaskTypeGenerationCompleted, data)
}
