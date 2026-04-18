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
	courseSessionDeletedPollInterval = 2 * time.Second
	courseSessionDeletedBatchSize    = 10
	courseSessionDeletedRetryAfter   = 30 * time.Second
)

type CourseSessionDeletedPublisher struct {
	tasks     taskRepository
	publisher eventPublisher
}

func NewCourseSessionDeletedPublisher(tasks taskRepository, publisher eventPublisher) *CourseSessionDeletedPublisher {
	return &CourseSessionDeletedPublisher{tasks: tasks, publisher: publisher}
}

func (w *CourseSessionDeletedPublisher) Run(ctx context.Context) error {
	slog.Info("course-session-deleted-publisher: запущен", "interval", courseSessionDeletedPollInterval)
	ticker := time.NewTicker(courseSessionDeletedPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("course-session-deleted-publisher: ошибка батча", "err", err)
			}
		}
	}
}

func (w *CourseSessionDeletedPublisher) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.CourseSessionDeleted, courseSessionDeletedBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("course-session-deleted-publisher: ошибка задачи", "task_id", t.ID, "err", err)
			if mfErr := w.tasks.MarkFailed(context.WithoutCancel(ctx), t.ID, err.Error(), courseSessionDeletedRetryAfter); mfErr != nil {
				slog.Error("course-session-deleted-publisher: не удалось сохранить ошибку задачи", "task_id", t.ID, "err", mfErr)
			}
		}
	}
	return nil
}

func (w *CourseSessionDeletedPublisher) processTask(ctx context.Context, t domain.Task) error {
	ctx, span := otel.Tracer("caesar").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.course_session_deleted_publisher")
	defer span.End()

	slog.InfoContext(ctx, "course-session-deleted-publisher: публикация", "task_id", t.ID)

	if err := w.publisher.Publish(ctx, natsinf.SubjectCourseSessionDeleted, t.Payload); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
