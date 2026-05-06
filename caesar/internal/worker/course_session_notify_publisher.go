package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"

	"caesar/internal/domain"
	natsinf "caesar/internal/infra/nats"
	"caesar/internal/infra/telemetry"
)

const (
	courseSessionNotifyPollInterval = 2 * time.Second
	courseSessionNotifyBatchSize    = 10
	courseSessionNotifyRetryAfter   = 30 * time.Second
)

type CourseSessionNotifyPublisher struct {
	tasks     taskRepository
	publisher *natsinf.Publisher
}

func NewCourseSessionNotifyPublisher(tasks taskRepository, publisher *natsinf.Publisher) *CourseSessionNotifyPublisher {
	return &CourseSessionNotifyPublisher{tasks: tasks, publisher: publisher}
}

func (w *CourseSessionNotifyPublisher) Run(ctx context.Context) error {
	slog.Info("course-session-notify-publisher: запущен", "interval", courseSessionNotifyPollInterval)
	ticker := time.NewTicker(courseSessionNotifyPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("course-session-notify-publisher: ошибка батча", "err", err)
			}
		}
	}
}

func (w *CourseSessionNotifyPublisher) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.CourseSessionNotify, courseSessionNotifyBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("course-session-notify-publisher: ошибка задачи", "task_id", t.ID, "err", err)
			_ = w.tasks.MarkFailed(ctx, t.ID, err.Error(), courseSessionNotifyRetryAfter)
		}
	}
	return nil
}

func (w *CourseSessionNotifyPublisher) processTask(ctx context.Context, t domain.Task) error {
	ctx, span := otel.Tracer("caesar").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.course_session_notify_publisher")
	defer span.End()

	if err := w.publisher.Publish(ctx, natsinf.SubjectCourseSessionNotify, t.Payload); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
