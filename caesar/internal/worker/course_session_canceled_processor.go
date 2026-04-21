package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"caesar/internal/domain"
	"caesar/internal/infra/telemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

const (
	courseSessionCanceledPollInterval = 2 * time.Second
	courseSessionCanceledBatchSize    = 10
	courseSessionCanceledRetryAfter   = 30 * time.Second
)

type courseSessionCanceledStore interface {
	FindItemByObjectID(ctx context.Context, objectID uuid.UUID) (*domain.CourseItem, error)
	DeleteItem(ctx context.Context, id uuid.UUID) error
	UpsertCourseDraft(ctx context.Context, quizTemplateID, courseID uuid.UUID, title string, payload json.RawMessage) (uuid.UUID, error)
}

type CourseSessionCanceledProcessor struct {
	tasks taskRepository
	store courseSessionCanceledStore
}

func NewCourseSessionCanceledProcessor(tasks taskRepository, store courseSessionCanceledStore) *CourseSessionCanceledProcessor {
	return &CourseSessionCanceledProcessor{tasks: tasks, store: store}
}

func (w *CourseSessionCanceledProcessor) Run(ctx context.Context) error {
	slog.Info("course-session-canceled-processor: запущен", "interval", courseSessionCanceledPollInterval)
	ticker := time.NewTicker(courseSessionCanceledPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("course-session-canceled-processor: ошибка батча", "err", err)
			}
		}
	}
}

func (w *CourseSessionCanceledProcessor) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.CourseSessionCanceled, courseSessionCanceledBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("course-session-canceled-processor: ошибка задачи", "task_id", t.ID, "err", err)
			if mfErr := w.tasks.MarkFailed(context.WithoutCancel(ctx), t.ID, err.Error(), courseSessionCanceledRetryAfter); mfErr != nil {
				slog.Error("course-session-canceled-processor: не удалось сохранить ошибку задачи", "task_id", t.ID, "err", mfErr)
			}
		}
	}
	return nil
}

type courseSessionCanceledPayload struct {
	SessionID      string          `json:"session_id"`
	QuizTemplateID string          `json:"quiz_template_id"`
	CourseID       *string         `json:"course_id,omitempty"`
	Title          string          `json:"title"`
	Payload        json.RawMessage `json:"payload,omitempty"`
}

func (w *CourseSessionCanceledProcessor) processTask(ctx context.Context, t domain.Task) error {
	var p courseSessionCanceledPayload
	if err := json.Unmarshal(t.Payload, &p); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("caesar").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.course_session_canceled_processor")
	defer span.End()

	sessionID, err := uuid.Parse(p.SessionID)
	if err != nil {
		return fmt.Errorf("parse session_id: %w", err)
	}

	slog.InfoContext(ctx, "course-session-canceled-processor: удаление элемента курса", "task_id", t.ID, "session_id", sessionID)

	item, err := w.store.FindItemByObjectID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("findItemByObjectID: %w", err)
	}
	if item != nil {
		if err := w.store.DeleteItem(ctx, item.ID); err != nil {
			return fmt.Errorf("deleteItem: %w", err)
		}
	}

	if p.CourseID != nil {
		quizTemplateID, err := uuid.Parse(p.QuizTemplateID)
		if err != nil {
			return fmt.Errorf("parse quiz_template_id: %w", err)
		}
		courseID, err := uuid.Parse(*p.CourseID)
		if err != nil {
			return fmt.Errorf("parse course_id: %w", err)
		}
		if _, err := w.store.UpsertCourseDraft(ctx, quizTemplateID, courseID, p.Title, p.Payload); err != nil {
			return fmt.Errorf("upsertCourseDraft: %w", err)
		}
	}

	return w.tasks.MarkDone(ctx, t.ID)
}
