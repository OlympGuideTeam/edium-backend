package worker

import (
	"context"
	"doorman/internal/domain"
	natsinf "doorman/internal/infra/nats"
	"doorman/internal/infra/telemetry"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
)

const (
	userCreatedPollInterval = 2 * time.Second
	userCreatedBatchSize    = 10
	userCreatedRetryAfter   = 30 * time.Second
)

type UserCreatedPublisher struct {
	tasks     taskRepository
	publisher *natsinf.Publisher
}

func NewUserCreatedPublisher(tasks taskRepository, publisher *natsinf.Publisher) *UserCreatedPublisher {
	return &UserCreatedPublisher{tasks: tasks, publisher: publisher}
}

func (w *UserCreatedPublisher) Run(ctx context.Context) error {
	slog.Info("user-created-publisher: запущен", "interval", userCreatedPollInterval)
	ticker := time.NewTicker(userCreatedPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("user-created-publisher: ошибка батча", "err", err)
			}
		}
	}
}

type userCreatedMsg struct {
	UserID  string `json:"user_id"`
	Phone   string `json:"phone"`
	Name    string `json:"name"`
	Surname string `json:"surname"`
}

func (w *UserCreatedPublisher) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.UserCreated, userCreatedBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("user-created-publisher: ошибка задачи", "task_id", t.ID, "err", err)
			_ = w.tasks.MarkFailed(ctx, t.ID, err.Error(), userCreatedRetryAfter)
		}
	}
	return nil
}

func (w *UserCreatedPublisher) processTask(ctx context.Context, t domain.Task) error {
	var payload userCreatedMsg
	if err := json.Unmarshal(t.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("doorman").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.user_created_publisher")
	defer span.End()

	slog.InfoContext(ctx, "user-created-publisher: публикация", "task_id", t.ID, "user_id", payload.UserID)

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	if err := w.publisher.Publish(ctx, natsinf.SubjectUserCreated, data); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
