package worker

import (
	"charon/internal/domain"
	natsinf "charon/internal/infra/nats"
	"charon/internal/infra/telemetry"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
)

const (
	publisherPollInterval = 2 * time.Second
	publisherBatchSize    = 10
	publisherRetryAfter   = 30 * time.Second
)

type CompletionPublisher struct {
	tasks     taskRepository
	publisher *natsinf.Publisher
}

func NewCompletionPublisher(tasks taskRepository, publisher *natsinf.Publisher) *CompletionPublisher {
	return &CompletionPublisher{tasks: tasks, publisher: publisher}
}

func (w *CompletionPublisher) Run(ctx context.Context) error {
	slog.Info("completion-publisher: started", "interval", publisherPollInterval)
	ticker := time.NewTicker(publisherPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("completion-publisher: batch error", "err", err)
			}
		}
	}
}

func (w *CompletionPublisher) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.CompletionCompleted, publisherBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("completion-publisher: task error", "task_id", t.ID, "err", err)
			if mfErr := w.tasks.MarkFailed(context.WithoutCancel(ctx), t.ID, err.Error(), publisherRetryAfter); mfErr != nil {
				slog.Error("completion-publisher: failed to save task error", "task_id", t.ID, "err", mfErr)
			}
		}
	}
	return nil
}

func (w *CompletionPublisher) processTask(ctx context.Context, t domain.Task) error {
	var payload domain.CompletionResponse
	if err := json.Unmarshal(t.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("charon").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.completion_publisher")
	defer span.End()

	slog.InfoContext(ctx, "completion-publisher: publishing", "task_id", t.ID, "request_id", payload.RequestID)

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	if err := w.publisher.Publish(ctx, natsinf.SubjectCompletionCompleted, data); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
