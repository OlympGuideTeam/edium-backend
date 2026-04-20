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

type courseItemPayload struct {
	Mode                 string     `json:"mode"`
	TotalTimeLimitSec    *int       `json:"total_time_limit_sec,omitempty"`
	QuestionTimeLimitSec *int       `json:"question_time_limit_sec,omitempty"`
	ShuffleQuestions     *bool      `json:"shuffle_questions,omitempty"`
	StartedAt            *time.Time `json:"started_at,omitempty"`
	FinishedAt           *time.Time `json:"finished_at,omitempty"`
}

const (
	courseSessionCreatedPollInterval = 2 * time.Second
	courseSessionCreatedBatchSize    = 10
	courseSessionCreatedRetryAfter   = 30 * time.Second
)

type CourseSessionCreatedProcessor struct {
	tasks taskRepository
	items courseItemStore
}

func NewCourseSessionCreatedProcessor(tasks taskRepository, items courseItemStore) *CourseSessionCreatedProcessor {
	return &CourseSessionCreatedProcessor{tasks: tasks, items: items}
}

func (w *CourseSessionCreatedProcessor) Run(ctx context.Context) error {
	slog.Info("course-session-created-processor: запущен", "interval", courseSessionCreatedPollInterval)
	ticker := time.NewTicker(courseSessionCreatedPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("course-session-created-processor: ошибка батча", "err", err)
			}
		}
	}
}

type courseSessionCreatedPayload struct {
	SessionID            string     `json:"session_id"`
	ModuleID             string     `json:"module_id"`
	Mode                 string     `json:"mode"`
	TotalTimeLimitSec    *int       `json:"total_time_limit_sec,omitempty"`
	QuestionTimeLimitSec *int       `json:"question_time_limit_sec,omitempty"`
	ShuffleQuestions     *bool      `json:"shuffle_questions,omitempty"`
	StartedAt            *time.Time `json:"started_at,omitempty"`
	FinishedAt           *time.Time `json:"finished_at,omitempty"`
}

func (w *CourseSessionCreatedProcessor) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.CourseSessionCreated, courseSessionCreatedBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("course-session-created-processor: ошибка задачи", "task_id", t.ID, "err", err)
			if mfErr := w.tasks.MarkFailed(context.WithoutCancel(ctx), t.ID, err.Error(), courseSessionCreatedRetryAfter); mfErr != nil {
				slog.Error("course-session-created-processor: не удалось сохранить ошибку задачи", "task_id", t.ID, "err", mfErr)
			}
		}
	}
	return nil
}

func (w *CourseSessionCreatedProcessor) processTask(ctx context.Context, t domain.Task) error {
	var p courseSessionCreatedPayload
	if err := json.Unmarshal(t.Payload, &p); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("caesar").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.course_session_created_processor")
	defer span.End()

	sessionID, err := uuid.Parse(p.SessionID)
	if err != nil {
		return fmt.Errorf("parse session_id: %w", err)
	}
	moduleID, err := uuid.Parse(p.ModuleID)
	if err != nil {
		return fmt.Errorf("parse module_id: %w", err)
	}

	slog.InfoContext(ctx, "course-session-created-processor: создание элемента", "task_id", t.ID, "session_id", sessionID)

	itemPayloadBytes, _ := json.Marshal(courseItemPayload{
		Mode:                 p.Mode,
		TotalTimeLimitSec:    p.TotalTimeLimitSec,
		QuestionTimeLimitSec: p.QuestionTimeLimitSec,
		ShuffleQuestions:     p.ShuffleQuestions,
		StartedAt:            p.StartedAt,
		FinishedAt:           p.FinishedAt,
	})

	if _, err := w.items.CreateItem(ctx, moduleID, sessionID, domain.CourseItemTypeQuiz, json.RawMessage(itemPayloadBytes)); err != nil {
		return fmt.Errorf("createItem: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
