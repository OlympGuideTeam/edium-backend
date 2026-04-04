package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"caesar/internal/domain"
	natsinf "caesar/internal/infra/nats"
	"caesar/internal/infra/telemetry"

	"go.opentelemetry.io/otel"
)

const (
	userDeletedPollInterval = 2 * time.Second
	userDeletedBatchSize    = 10
	userDeletedRetryAfter   = 30 * time.Second
)

type eventPublisher interface {
	Publish(ctx context.Context, subject string, data []byte) error
}

type UserDeletedPublisher struct {
	tasks     taskRepository
	publisher eventPublisher
}

func NewUserDeletedPublisher(tasks taskRepository, publisher eventPublisher) *UserDeletedPublisher {
	return &UserDeletedPublisher{tasks: tasks, publisher: publisher}
}

func (w *UserDeletedPublisher) Run(ctx context.Context) error {
	slog.Info("user-deleted-publisher: запущен", "interval", userDeletedPollInterval)
	ticker := time.NewTicker(userDeletedPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("user-deleted-publisher: ошибка батча", "err", err)
			}
		}
	}
}

func (w *UserDeletedPublisher) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.UserDeleted, userDeletedBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("user-deleted-publisher: ошибка задачи", "task_id", t.ID, "err", err)
			if mfErr := w.tasks.MarkFailed(context.WithoutCancel(ctx), t.ID, err.Error(), userDeletedRetryAfter); mfErr != nil {
				slog.Error("user-deleted-publisher: не удалось сохранить ошибку задачи", "task_id", t.ID, "err", mfErr)
			}
		}
	}
	return nil
}

func (w *UserDeletedPublisher) processTask(ctx context.Context, t domain.Task) error {
	ctx, span := otel.Tracer("caesar").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.user_deleted_publisher")
	defer span.End()

	slog.InfoContext(ctx, "user-deleted-publisher: публикация", "task_id", t.ID)

	if err := w.publisher.Publish(ctx, natsinf.SubjectUserDeleted, t.Payload); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
