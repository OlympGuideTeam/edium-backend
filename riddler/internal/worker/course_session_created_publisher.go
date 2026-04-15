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
	courseSessionCreatedPollInterval = 2 * time.Second
	courseSessionCreatedBatchSize    = 10
	courseSessionCreatedRetryAfter   = 30 * time.Second
)

type CourseSessionCreatedPublisher struct {
	tasks     taskRepository
	publisher natsPublisher
}

func NewCourseSessionCreatedPublisher(tasks taskRepository, publisher natsPublisher) *CourseSessionCreatedPublisher {
	return &CourseSessionCreatedPublisher{tasks: tasks, publisher: publisher}
}

func (w *CourseSessionCreatedPublisher) Run(ctx context.Context) error {
	slog.Info("course-session-created-publisher: запущен")
	ticker := time.NewTicker(courseSessionCreatedPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("course-session-created-publisher: ошибка батча", "err", err)
			}
		}
	}
}

func (w *CourseSessionCreatedPublisher) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.TaskTypeCourseSessionCreatedPublisher, courseSessionCreatedBatchSize)
	if err != nil {
		return fmt.Errorf("claim pending: %w", err)
	}
	for i := range tasks {
		if err := w.processTask(ctx, tasks[i]); err != nil {
			slog.Error("course-session-created-publisher: ошибка задачи", "task_id", tasks[i].ID, "err", err)
			_ = w.tasks.MarkFailed(ctx, tasks[i].ID, err.Error(), courseSessionCreatedRetryAfter)
		}
	}
	return nil
}

func (w *CourseSessionCreatedPublisher) processTask(ctx context.Context, t domain.Task) error {
	var msg struct {
		SessionID string `json:"session_id"`
		ModuleID  string `json:"module_id"`
	}
	if err := json.Unmarshal(t.Payload, &msg); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("riddler").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.course_session_created_publisher")
	defer span.End()

	if err := w.publisher.Publish(ctx, natsinf.SubjectCourseSessionCreated, t.Payload); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
