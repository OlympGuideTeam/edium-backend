package worker

import (
	"charon/internal/domain"
	"charon/internal/infra/telemetry"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

const (
	completionPollInterval = 2 * time.Second
	completionBatchSize    = 10
	completionRetryAfter   = 30 * time.Second
)

type taskRepository interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
	ClaimPending(ctx context.Context, taskType domain.TaskType, limit int) ([]domain.Task, error)
	MarkDone(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string, retryAfter time.Duration) error
}

type completionService interface {
	ProcessCompletion(ctx context.Context, req domain.CompletionRequest) error
}

type CompletionProcessor struct {
	tasks   taskRepository
	service completionService
}

func NewCompletionProcessor(tasks taskRepository, service completionService) *CompletionProcessor {
	return &CompletionProcessor{tasks: tasks, service: service}
}

func (w *CompletionProcessor) Run(ctx context.Context) error {
	slog.Info("completion-processor: started", "interval", completionPollInterval)
	ticker := time.NewTicker(completionPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("completion-processor: batch error", "err", err)
			}
		}
	}
}

func (w *CompletionProcessor) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.CompletionRequested, completionBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("completion-processor: task error", "task_id", t.ID, "err", err)
			if mfErr := w.tasks.MarkFailed(context.WithoutCancel(ctx), t.ID, err.Error(), completionRetryAfter); mfErr != nil {
				slog.Error("completion-processor: failed to save task error", "task_id", t.ID, "err", mfErr)
			}
		}
	}
	return nil
}

func (w *CompletionProcessor) processTask(ctx context.Context, t domain.Task) error {
	var req domain.CompletionRequest
	if err := json.Unmarshal(t.Payload, &req); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("charon").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.completion_processor")
	defer span.End()

	slog.InfoContext(ctx, "completion-processor: processing",
		"task_id", t.ID, "request_id", req.RequestID, "model", req.Model)

	if err := w.service.ProcessCompletion(ctx, req); err != nil {
		return err
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
