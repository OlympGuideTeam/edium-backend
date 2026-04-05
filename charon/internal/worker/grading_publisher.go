package worker

import (
	"charon/internal/domain"
	natsinf "charon/internal/infra/nats"
	"charon/internal/infra/telemetry"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
)

const (
	gradingPublisherPollInterval = 2 * time.Second
	gradingPublisherBatchSize    = 10
	gradingPublisherRetryAfter   = 30 * time.Second
)

type GradingPublisher struct {
	tasks     taskRepository
	publisher *natsinf.Publisher
}

func NewGradingPublisher(tasks taskRepository, publisher *natsinf.Publisher) *GradingPublisher {
	return &GradingPublisher{tasks: tasks, publisher: publisher}
}

func (w *GradingPublisher) Run(ctx context.Context) error {
	slog.Info("grading-publisher: started", "interval", gradingPublisherPollInterval)
	ticker := time.NewTicker(gradingPublisherPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("grading-publisher: batch error", "err", err)
			}
		}
	}
}

func (w *GradingPublisher) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.QuizGradeCompleted, gradingPublisherBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("grading-publisher: task error", "task_id", t.ID, "err", err)
			if mfErr := w.tasks.MarkFailed(context.WithoutCancel(ctx), t.ID, err.Error(), gradingPublisherRetryAfter); mfErr != nil {
				slog.Error("grading-publisher: failed to save task error", "task_id", t.ID, "err", mfErr)
			}
		}
	}
	return nil
}

func (w *GradingPublisher) processTask(ctx context.Context, t domain.Task) error {
	var payload domain.QuizGradeResponse
	if err := json.Unmarshal(t.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("charon").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.grading_publisher")
	defer span.End()

	slog.InfoContext(ctx, "grading-publisher: publishing",
		"task_id", t.ID, "request_id", payload.RequestID, "grades", len(payload.Grades))

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	if err := w.publisher.Publish(ctx, natsinf.SubjectQuizGradeCompleted, data); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
