package worker

import (
	"context"
	"doorman/internal/domain"
	"doorman/internal/infra/telemetry"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
)

const (
	userDeletedPollInterval = 2 * time.Second
	userDeletedBatchSize    = 10
	userDeletedRetryAfter   = 30 * time.Second
)

type identityStatusUpdater interface {
	UpdateStatus(ctx context.Context, userID string, status domain.IdentityStatus) error
}

type userTokensCleaner interface {
	DeleteUserTokens(ctx context.Context, userID string) error
}

type UserDeletedProcessor struct {
	tasks    taskRepository
	identity identityStatusUpdater
	tokens   userTokensCleaner
}

func NewUserDeletedProcessor(
	tasks taskRepository,
	identity identityStatusUpdater,
	tokens userTokensCleaner,
) *UserDeletedProcessor {
	return &UserDeletedProcessor{tasks: tasks, identity: identity, tokens: tokens}
}

func (w *UserDeletedProcessor) Run(ctx context.Context) error {
	slog.Info("user-deleted-processor: запущен", "interval", userDeletedPollInterval)
	ticker := time.NewTicker(userDeletedPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("user-deleted-processor: ошибка батча", "err", err)
			}
		}
	}
}

func (w *UserDeletedProcessor) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.UserDeleted, userDeletedBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("user-deleted-processor: ошибка задачи", "task_id", t.ID, "err", err)
			_ = w.tasks.MarkFailed(ctx, t.ID, err.Error(), userDeletedRetryAfter)
		}
	}
	return nil
}

func (w *UserDeletedProcessor) processTask(ctx context.Context, t domain.Task) error {
	var payload struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(t.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("doorman").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.user_deleted_processor")
	defer span.End()

	slog.InfoContext(ctx, "user-deleted-processor: обработка", "task_id", t.ID, "user_id", payload.UserID)

	if err := w.identity.UpdateStatus(ctx, payload.UserID, domain.IdentityStatusDeleted); err != nil {
		return fmt.Errorf("UpdateStatus: %w", err)
	}

	if err := w.tokens.DeleteUserTokens(ctx, payload.UserID); err != nil {
		return fmt.Errorf("DeleteUserTokens: %w", err)
	}

	return w.tasks.MarkDone(ctx, t.ID)
}
