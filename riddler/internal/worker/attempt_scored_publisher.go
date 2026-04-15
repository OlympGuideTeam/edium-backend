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
	attemptScoredPollInterval = 2 * time.Second
	attemptScoredBatchSize    = 10
	attemptScoredRetryAfter   = 30 * time.Second
)

type AttemptScoredPublisher struct {
	tasks     taskRepository
	publisher natsPublisher
}

func NewAttemptScoredPublisher(tasks taskRepository, publisher natsPublisher) *AttemptScoredPublisher {
	return &AttemptScoredPublisher{tasks: tasks, publisher: publisher}
}

func (w *AttemptScoredPublisher) Run(ctx context.Context) error {
	slog.Info("attempt-scored-publisher: запущен")
	ticker := time.NewTicker(attemptScoredPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("attempt-scored-publisher: ошибка батча", "err", err)
			}
		}
	}
}

func (w *AttemptScoredPublisher) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.TaskTypeAttemptScoredPublisher, attemptScoredBatchSize)
	if err != nil {
		return fmt.Errorf("claim pending: %w", err)
	}
	for i := range tasks {
		if err := w.processTask(ctx, tasks[i]); err != nil {
			slog.Error("attempt-scored-publisher: ошибка задачи", "task_id", tasks[i].ID, "err", err)
			_ = w.tasks.MarkFailed(ctx, tasks[i].ID, err.Error(), attemptScoredRetryAfter)
		}
	}
	return nil
}

func (w *AttemptScoredPublisher) processTask(ctx context.Context, t domain.Task) error {
	var msg struct {
		AttemptID  string  `json:"attempt_id"`
		SessionID  string  `json:"session_id"`
		UserID     string  `json:"user_id"`
		TotalScore float64 `json:"total_score"`
	}
	if err := json.Unmarshal(t.Payload, &msg); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("riddler").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.attempt_scored_publisher")
	defer span.End()

	if err := w.publisher.Publish(ctx, natsinf.SubjectAttemptScored, t.Payload); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
