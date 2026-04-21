package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"

	"riddler/internal/domain"
	natsinf "riddler/internal/infra/nats"
	"riddler/internal/infra/telemetry"
)

const (
	courseSessionCanceledPollInterval = 2 * time.Second
	courseSessionCanceledBatchSize    = 10
	courseSessionCanceledRetryAfter   = 30 * time.Second
)

type CourseSessionCanceledPublisher struct {
	tasks     taskRepository
	publisher natsPublisher
}

func NewCourseSessionCanceledPublisher(tasks taskRepository, publisher natsPublisher) *CourseSessionCanceledPublisher {
	return &CourseSessionCanceledPublisher{tasks: tasks, publisher: publisher}
}

func (w *CourseSessionCanceledPublisher) Run(ctx context.Context) error {
	slog.Info("course-session-canceled-publisher: запущен")
	ticker := time.NewTicker(courseSessionCanceledPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("course-session-canceled-publisher: ошибка батча", "err", err)
			}
		}
	}
}

func (w *CourseSessionCanceledPublisher) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.TaskTypeCourseSessionCanceledPublisher, courseSessionCanceledBatchSize)
	if err != nil {
		return fmt.Errorf("claim pending: %w", err)
	}
	for i := range tasks {
		if err := w.processTask(ctx, tasks[i]); err != nil {
			slog.Error("course-session-canceled-publisher: ошибка задачи", "task_id", tasks[i].ID, "err", err)
			_ = w.tasks.MarkFailed(ctx, tasks[i].ID, err.Error(), courseSessionCanceledRetryAfter)
		}
	}
	return nil
}

func (w *CourseSessionCanceledPublisher) processTask(ctx context.Context, t domain.Task) error {
	var msg struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(t.Payload, &msg); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("riddler").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.course_session_canceled_publisher")
	defer span.End()

	if err := w.publisher.Publish(ctx, natsinf.SubjectCourseSessionCanceled, t.Payload); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
