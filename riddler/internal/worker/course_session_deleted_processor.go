package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"riddler/internal/domain"
	"riddler/internal/infra/telemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

const (
	courseSessionDeletedPollInterval = 2 * time.Second
	courseSessionDeletedBatchSize    = 10
	courseSessionDeletedRetryAfter   = 30 * time.Second
)

type courseSessionDeletedTaskRepository interface {
	ClaimPending(ctx context.Context, taskType domain.TaskType, limit int) ([]domain.Task, error)
	MarkDone(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string, retryAfter time.Duration) error
}

type sessionDeleter interface {
	Delete(ctx context.Context, id uuid.UUID) error
}

type CourseSessionDeletedProcessor struct {
	tasks    courseSessionDeletedTaskRepository
	sessions sessionDeleter
}

func NewCourseSessionDeletedProcessor(tasks courseSessionDeletedTaskRepository, sessions sessionDeleter) *CourseSessionDeletedProcessor {
	return &CourseSessionDeletedProcessor{tasks: tasks, sessions: sessions}
}

func (w *CourseSessionDeletedProcessor) Run(ctx context.Context) error {
	slog.Info("course-session-deleted-processor: запущен", "interval", courseSessionDeletedPollInterval)
	ticker := time.NewTicker(courseSessionDeletedPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("course-session-deleted-processor: ошибка батча", "err", err)
			}
		}
	}
}

type courseSessionDeletedPayload struct {
	SessionID uuid.UUID `json:"session_id"`
}

func (w *CourseSessionDeletedProcessor) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.TaskTypeCourseSessionDeleted, courseSessionDeletedBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("course-session-deleted-processor: ошибка задачи", "task_id", t.ID, "err", err)
			if mfErr := w.tasks.MarkFailed(context.WithoutCancel(ctx), t.ID, err.Error(), courseSessionDeletedRetryAfter); mfErr != nil {
				slog.Error("course-session-deleted-processor: не удалось сохранить ошибку задачи", "task_id", t.ID, "err", mfErr)
			}
		}
	}
	return nil
}

func (w *CourseSessionDeletedProcessor) processTask(ctx context.Context, t domain.Task) error {
	var p courseSessionDeletedPayload
	if err := json.Unmarshal(t.Payload, &p); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("riddler").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.course_session_deleted_processor")
	defer span.End()

	slog.InfoContext(ctx, "course-session-deleted-processor: удаление сессии", "task_id", t.ID, "session_id", p.SessionID)

	if err := w.sessions.Delete(ctx, p.SessionID); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
