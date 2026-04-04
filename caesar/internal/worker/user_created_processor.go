package worker

import (
	"caesar/internal/domain"
	"caesar/internal/infra/telemetry"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

const (
	userCreatedPollInterval = 2 * time.Second
	userCreatedBatchSize    = 10
	userCreatedRetryAfter   = 30 * time.Second
)

type userStore interface {
	Create(ctx context.Context, u domain.User) error
}

type UserCreatedProcessor struct {
	tasks taskRepository
	users userStore
}

func NewUserCreatedProcessor(tasks taskRepository, users userStore) *UserCreatedProcessor {
	return &UserCreatedProcessor{tasks: tasks, users: users}
}

func (w *UserCreatedProcessor) Run(ctx context.Context) error {
	slog.Info("user-created-processor: запущен", "interval", userCreatedPollInterval)
	ticker := time.NewTicker(userCreatedPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("user-created-processor: ошибка батча", "err", err)
			}
		}
	}
}

type userCreatedPayload struct {
	UserID  string `json:"user_id"`
	Phone   string `json:"phone"`
	Name    string `json:"name"`
	Surname string `json:"surname"`
}

func (w *UserCreatedProcessor) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.UserCreated, userCreatedBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("user-created-processor: ошибка задачи", "task_id", t.ID, "err", err)
			if mfErr := w.tasks.MarkFailed(context.WithoutCancel(ctx), t.ID, err.Error(), userCreatedRetryAfter); mfErr != nil {
				slog.Error("user-created-processor: не удалось сохранить ошибку задачи", "task_id", t.ID, "err", mfErr)
			}
		}
	}
	return nil
}

func (w *UserCreatedProcessor) processTask(ctx context.Context, t domain.Task) error {
	var p userCreatedPayload
	if err := json.Unmarshal(t.Payload, &p); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("caesar").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.user_created_processor")
	defer span.End()

	slog.InfoContext(ctx, "user-created-processor: создание пользователя", "task_id", t.ID, "user_id", p.UserID)

	userID, err := uuid.Parse(p.UserID)
	if err != nil {
		return fmt.Errorf("parse user_id: %w", err)
	}

	u := domain.User{
		ID:      userID,
		Name:    p.Name,
		Surname: p.Surname,
		Phone:   p.Phone,
		Status:  domain.UserStatusActive,
	}

	if err := w.users.Create(ctx, u); err != nil {
		return fmt.Errorf("create: %w", err)
	}

	return w.tasks.MarkDone(ctx, t.ID)
}
