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
	quizTemplateAttachedPollInterval = 2 * time.Second
	quizTemplateAttachedBatchSize    = 10
	quizTemplateAttachedRetryAfter   = 30 * time.Second
)

type QuizTemplateAttachedProcessor struct {
	tasks  taskRepository
	drafts courseDraftStore
}

func NewQuizTemplateAttachedProcessor(tasks taskRepository, drafts courseDraftStore) *QuizTemplateAttachedProcessor {
	return &QuizTemplateAttachedProcessor{tasks: tasks, drafts: drafts}
}

func (w *QuizTemplateAttachedProcessor) Run(ctx context.Context) error {
	slog.Info("quiz-template-attached-processor: запущен", "interval", quizTemplateAttachedPollInterval)
	ticker := time.NewTicker(quizTemplateAttachedPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("quiz-template-attached-processor: ошибка батча", "err", err)
			}
		}
	}
}

type quizTemplateAttachedPayload struct {
	QuizTemplateID string          `json:"quiz_template_id"`
	CourseID       string          `json:"course_id"`
	Title          string          `json:"title"`
	Payload        json.RawMessage `json:"payload,omitempty"`
}

func (w *QuizTemplateAttachedProcessor) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.QuizTemplateAttached, quizTemplateAttachedBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("quiz-template-attached-processor: ошибка задачи", "task_id", t.ID, "err", err)
			if mfErr := w.tasks.MarkFailed(context.WithoutCancel(ctx), t.ID, err.Error(), quizTemplateAttachedRetryAfter); mfErr != nil {
				slog.Error("quiz-template-attached-processor: не удалось сохранить ошибку задачи", "task_id", t.ID, "err", mfErr)
			}
		}
	}
	return nil
}

func (w *QuizTemplateAttachedProcessor) processTask(ctx context.Context, t domain.Task) error {
	var p quizTemplateAttachedPayload
	if err := json.Unmarshal(t.Payload, &p); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("caesar").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.quiz_template_attached_processor")
	defer span.End()

	quizTemplateID, err := uuid.Parse(p.QuizTemplateID)
	if err != nil {
		return fmt.Errorf("parse quiz_template_id: %w", err)
	}
	courseID, err := uuid.Parse(p.CourseID)
	if err != nil {
		return fmt.Errorf("parse course_id: %w", err)
	}

	slog.InfoContext(ctx, "quiz-template-attached-processor: upsert черновика", "task_id", t.ID, "quiz_template_id", quizTemplateID)

	if _, err := w.drafts.UpsertCourseDraft(ctx, quizTemplateID, courseID, p.Title, p.Payload); err != nil {
		return fmt.Errorf("upsertCourseDraft: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
