package worker

import (
	"charon/internal/domain"
	"charon/internal/infra/telemetry"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
)

const (
	gradingPollInterval = 2 * time.Second
	gradingBatchSize    = 5
	gradingRetryAfter   = 30 * time.Second
)

type gradingService interface {
	ProcessGrade(ctx context.Context, req domain.QuizGradeRequest) error
}

type GradingProcessor struct {
	tasks   taskRepository
	service gradingService
}

func NewGradingProcessor(tasks taskRepository, service gradingService) *GradingProcessor {
	return &GradingProcessor{tasks: tasks, service: service}
}

func (w *GradingProcessor) Run(ctx context.Context) error {
	slog.Info("grading-processor: started", "interval", gradingPollInterval)
	ticker := time.NewTicker(gradingPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("grading-processor: batch error", "err", err)
			}
		}
	}
}

func (w *GradingProcessor) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.QuizGradeRequested, gradingBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("grading-processor: task error", "task_id", t.ID, "err", err)
			if mfErr := w.tasks.MarkFailed(context.WithoutCancel(ctx), t.ID, err.Error(), gradingRetryAfter); mfErr != nil {
				slog.Error("grading-processor: failed to save task error", "task_id", t.ID, "err", mfErr)
			}
		}
	}
	return nil
}

func (w *GradingProcessor) processTask(ctx context.Context, t domain.Task) error {
	var req domain.QuizGradeRequest
	if err := json.Unmarshal(t.Payload, &req); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("charon").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.grading_processor")
	defer span.End()

	slog.InfoContext(ctx, "grading-processor: processing",
		"task_id", t.ID, "request_id", req.RequestID, "answers", len(req.Answers))

	if err := w.service.ProcessGrade(ctx, req); err != nil {
		return err
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
