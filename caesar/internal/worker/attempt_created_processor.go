package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"caesar/internal/domain"
	"caesar/internal/infra/telemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

const (
	attemptCreatedPollInterval = 2 * time.Second
	attemptCreatedBatchSize    = 10
	attemptCreatedRetryAfter   = 30 * time.Second
)

type AttemptCreatedProcessor struct {
	tasks taskRepository
	items courseItemStore
}

func NewAttemptCreatedProcessor(tasks taskRepository, items courseItemStore) *AttemptCreatedProcessor {
	return &AttemptCreatedProcessor{tasks: tasks, items: items}
}

func (w *AttemptCreatedProcessor) Run(ctx context.Context) error {
	slog.Info("attempt-created-processor: запущен", "interval", attemptCreatedPollInterval)
	ticker := time.NewTicker(attemptCreatedPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("attempt-created-processor: ошибка батча", "err", err)
			}
		}
	}
}

type attemptCreatedPayload struct {
	AttemptID string `json:"attempt_id"`
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
}

func (w *AttemptCreatedProcessor) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.AttemptCreated, attemptCreatedBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("attempt-created-processor: ошибка задачи", "task_id", t.ID, "err", err)
			if mfErr := w.tasks.MarkFailed(context.WithoutCancel(ctx), t.ID, err.Error(), attemptCreatedRetryAfter); mfErr != nil {
				slog.Error("attempt-created-processor: не удалось сохранить ошибку задачи", "task_id", t.ID, "err", mfErr)
			}
		}
	}
	return nil
}

func (w *AttemptCreatedProcessor) processTask(ctx context.Context, t domain.Task) error {
	var p attemptCreatedPayload
	if err := json.Unmarshal(t.Payload, &p); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("caesar").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.attempt_created_processor")
	defer span.End()

	attemptID, err := uuid.Parse(p.AttemptID)
	if err != nil {
		return fmt.Errorf("parse attempt_id: %w", err)
	}
	sessionID, err := uuid.Parse(p.SessionID)
	if err != nil {
		return fmt.Errorf("parse session_id: %w", err)
	}
	userID, err := uuid.Parse(p.UserID)
	if err != nil {
		return fmt.Errorf("parse user_id: %w", err)
	}

	item, err := w.items.FindItemByObjectID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("findItemByObjectID: %w", err)
	}
	if item == nil {
		slog.WarnContext(ctx, "attempt-created-processor: элемент курса не найден", "session_id", sessionID)
		return w.tasks.MarkDone(ctx, t.ID)
	}

	slog.InfoContext(ctx, "attempt-created-processor: обновление прогресса", "task_id", t.ID, "item_id", item.ID)

	if err := w.items.UpsertProgress(ctx, item.ID, userID, attemptID); err != nil {
		return fmt.Errorf("upsertProgress: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
