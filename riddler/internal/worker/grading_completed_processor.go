package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"

	"riddler/internal/domain"
	"riddler/internal/infra/telemetry"
)

const (
	gradingCompletedPollInterval = 2 * time.Second
	gradingCompletedBatchSize    = 5
	gradingCompletedRetryAfter   = 30 * time.Second
)

type gradingCompletedTaskRepository interface {
	ClaimPending(ctx context.Context, taskType domain.TaskType, limit int) ([]domain.Task, error)
	MarkDone(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string, retryAfter time.Duration) error
}

type llmGradeApplier interface {
	ApplyLLMGrade(ctx context.Context, evalID uuid.UUID, score0to10 int, feedback *string) error
}

type GradingCompletedProcessor struct {
	tasks   gradingCompletedTaskRepository
	service llmGradeApplier
}

func NewGradingCompletedProcessor(tasks gradingCompletedTaskRepository, service llmGradeApplier) *GradingCompletedProcessor {
	return &GradingCompletedProcessor{tasks: tasks, service: service}
}

func (w *GradingCompletedProcessor) Run(ctx context.Context) error {
	slog.Info("grading-completed-processor: запущен")
	ticker := time.NewTicker(gradingCompletedPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("grading-completed-processor: ошибка батча", "err", err)
			}
		}
	}
}

func (w *GradingCompletedProcessor) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.TaskTypeGradingCompleted, gradingCompletedBatchSize)
	if err != nil {
		return fmt.Errorf("claim pending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("grading-completed-processor: ошибка задачи", "task_id", t.ID, "err", err)
			if mfErr := w.tasks.MarkFailed(context.WithoutCancel(ctx), t.ID, err.Error(), gradingCompletedRetryAfter); mfErr != nil {
				slog.Error("grading-completed-processor: не удалось сохранить ошибку", "task_id", t.ID, "err", mfErr)
			}
		}
	}
	return nil
}

type charonGradeResponse struct {
	RequestID string `json:"request_id"`
	Grades    []struct {
		StudentID string `json:"student_id"`
		Score     int    `json:"score"`
		Comment   string `json:"comment"`
	} `json:"grades"`
	Error string `json:"error"`
}

func (w *GradingCompletedProcessor) processTask(ctx context.Context, t domain.Task) error {
	var resp charonGradeResponse
	if err := json.Unmarshal(t.Payload, &resp); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("riddler").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.grading_completed_processor")
	defer span.End()

	if resp.Error != "" {
		slog.WarnContext(ctx, "grading-completed-processor: charon вернул ошибку", "request_id", resp.RequestID, "error", resp.Error)
		return w.tasks.MarkDone(ctx, t.ID)
	}

	for _, grade := range resp.Grades {
		evalID, err := uuid.Parse(grade.StudentID)
		if err != nil {
			return fmt.Errorf("parse eval_id %q: %w", grade.StudentID, err)
		}
		comment := grade.Comment
		if err := w.service.ApplyLLMGrade(ctx, evalID, grade.Score, &comment); err != nil {
			return fmt.Errorf("apply grade for eval %s: %w", evalID, err)
		}
	}

	return w.tasks.MarkDone(ctx, t.ID)
}
