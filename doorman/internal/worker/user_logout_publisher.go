package worker

import (
	"context"
	"doorman/internal/domain"
	natsinf "doorman/internal/infra/nats"
	"doorman/internal/infra/telemetry"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
)

const (
	userLogoutPollInterval = 2 * time.Second
	userLogoutBatchSize    = 10
	userLogoutRetryAfter   = 30 * time.Second
)

type UserLogoutPublisher struct {
	tasks     taskRepository
	publisher *natsinf.Publisher
}

func NewUserLogoutPublisher(tasks taskRepository, publisher *natsinf.Publisher) *UserLogoutPublisher {
	return &UserLogoutPublisher{tasks: tasks, publisher: publisher}
}

func (w *UserLogoutPublisher) Run(ctx context.Context) error {
	slog.Info("user-logout-publisher: запущен", "interval", userLogoutPollInterval)
	ticker := time.NewTicker(userLogoutPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("user-logout-publisher: ошибка батча", "err", err)
			}
		}
	}
}

func (w *UserLogoutPublisher) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.UserLogout, userLogoutBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("user-logout-publisher: ошибка задачи", "task_id", t.ID, "err", err)
			_ = w.tasks.MarkFailed(ctx, t.ID, err.Error(), userLogoutRetryAfter)
		}
	}
	return nil
}

func (w *UserLogoutPublisher) processTask(ctx context.Context, t domain.Task) error {
	ctx, span := otel.Tracer("doorman").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.user_logout_publisher")
	defer span.End()

	slog.InfoContext(ctx, "user-logout-publisher: публикация", "task_id", t.ID)

	if err := w.publisher.Publish(ctx, natsinf.SubjectUserLogout, t.Payload); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
