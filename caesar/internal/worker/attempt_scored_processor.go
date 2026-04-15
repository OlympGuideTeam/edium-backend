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
	attemptScoredPollInterval = 2 * time.Second
	attemptScoredBatchSize    = 10
	attemptScoredRetryAfter   = 30 * time.Second
)

type AttemptScoredProcessor struct {
	tasks taskRepository
	items courseItemStore
}

func NewAttemptScoredProcessor(tasks taskRepository, items courseItemStore) *AttemptScoredProcessor {
	return &AttemptScoredProcessor{tasks: tasks, items: items}
}

func (w *AttemptScoredProcessor) Run(ctx context.Context) error {
	slog.Info("attempt-scored-processor: запущен", "interval", attemptScoredPollInterval)
	ticker := time.NewTicker(attemptScoredPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("attempt-scored-processor: ошибка батча", "err", err)
			}
		}
	}
}

type attemptScoredPayload struct {
	AttemptID  string  `json:"attempt_id"`
	SessionID  string  `json:"session_id"`
	UserID     string  `json:"user_id"`
	TotalScore float64 `json:"total_score"`
}

func (w *AttemptScoredProcessor) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.AttemptScored, attemptScoredBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("attempt-scored-processor: ошибка задачи", "task_id", t.ID, "err", err)
			if mfErr := w.tasks.MarkFailed(context.WithoutCancel(ctx), t.ID, err.Error(), attemptScoredRetryAfter); mfErr != nil {
				slog.Error("attempt-scored-processor: не удалось сохранить ошибку задачи", "task_id", t.ID, "err", mfErr)
			}
		}
	}
	return nil
}

func (w *AttemptScoredProcessor) processTask(ctx context.Context, t domain.Task) error {
	var p attemptScoredPayload
	if err := json.Unmarshal(t.Payload, &p); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("caesar").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.attempt_scored_processor")
	defer span.End()

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
		slog.WarnContext(ctx, "attempt-scored-processor: элемент курса не найден", "session_id", sessionID)
		return w.tasks.MarkDone(ctx, t.ID)
	}

	slog.InfoContext(ctx, "attempt-scored-processor: обновление оценки", "task_id", t.ID, "item_id", item.ID, "score", p.TotalScore)

	if err := w.items.UpdateProgressScore(ctx, item.ID, userID, p.TotalScore); err != nil {
		return fmt.Errorf("updateProgressScore: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
