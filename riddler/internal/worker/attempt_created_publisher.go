package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"

	"riddler/internal/domain"
	natsinf "riddler/internal/infra/nats"
	"riddler/internal/infra/telemetry"
)

const (
	attemptCreatedPollInterval = 2 * time.Second
	attemptCreatedBatchSize    = 10
	attemptCreatedRetryAfter   = 30 * time.Second
)

type AttemptCreatedPublisher struct {
	tasks     taskRepository
	publisher natsPublisher
}

func NewAttemptCreatedPublisher(tasks taskRepository, publisher natsPublisher) *AttemptCreatedPublisher {
	return &AttemptCreatedPublisher{tasks: tasks, publisher: publisher}
}

func (w *AttemptCreatedPublisher) Run(ctx context.Context) error {
	slog.Info("attempt-created-publisher: запущен")
	ticker := time.NewTicker(attemptCreatedPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("attempt-created-publisher: ошибка батча", "err", err)
			}
		}
	}
}

func (w *AttemptCreatedPublisher) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.TaskTypeAttemptCreatedPublisher, attemptCreatedBatchSize)
	if err != nil {
		return fmt.Errorf("claim pending: %w", err)
	}
	for i := range tasks {
		if err := w.processTask(ctx, tasks[i]); err != nil {
			slog.Error("attempt-created-publisher: ошибка задачи", "task_id", tasks[i].ID, "err", err)
			_ = w.tasks.MarkFailed(ctx, tasks[i].ID, err.Error(), attemptCreatedRetryAfter)
		}
	}
	return nil
}

func (w *AttemptCreatedPublisher) processTask(ctx context.Context, t domain.Task) error {
	var msg struct {
		AttemptID string `json:"attempt_id"`
		SessionID string `json:"session_id"`
		UserID    string `json:"user_id"`
	}
	if err := json.Unmarshal(t.Payload, &msg); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("riddler").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.attempt_created_publisher")
	defer span.End()

	if err := w.publisher.Publish(ctx, natsinf.SubjectAttemptCreated, t.Payload); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
