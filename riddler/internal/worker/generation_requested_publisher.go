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
	natsinf "riddler/internal/infra/nats"
	"riddler/internal/infra/telemetry"
)

const (
	generationRequestedPollInterval = 2 * time.Second
	generationRequestedBatchSize    = 10
	generationRequestedRetryAfter   = 30 * time.Second
)

type generationRequestedTaskRepository interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
	ClaimPending(ctx context.Context, taskType domain.TaskType, limit int) ([]domain.Task, error)
	MarkDone(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string, retryAfter time.Duration) error
}

type generationNATSPublisher interface {
	Publish(ctx context.Context, subject string, data []byte) error
}

type GenerationRequestedPublisher struct {
	tasks     generationRequestedTaskRepository
	publisher generationNATSPublisher
}

func NewGenerationRequestedPublisher(tasks generationRequestedTaskRepository, publisher generationNATSPublisher) *GenerationRequestedPublisher {
	return &GenerationRequestedPublisher{tasks: tasks, publisher: publisher}
}

func (w *GenerationRequestedPublisher) Run(ctx context.Context) error {
	slog.Info("generation-requested-publisher: запущен", "interval", generationRequestedPollInterval)
	ticker := time.NewTicker(generationRequestedPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("generation-requested-publisher: ошибка батча", "err", err)
			}
		}
	}
}

func (w *GenerationRequestedPublisher) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.TaskTypeGenerationRequestedPublisher, generationRequestedBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("generation-requested-publisher: ошибка задачи", "task_id", t.ID, "err", err)
			if mfErr := w.tasks.MarkFailed(context.WithoutCancel(ctx), t.ID, err.Error(), generationRequestedRetryAfter); mfErr != nil {
				slog.Error("generation-requested-publisher: не удалось сохранить ошибку задачи", "task_id", t.ID, "err", mfErr)
			}
		}
	}
	return nil
}

type generationRequestedMsg struct {
	JobID  uuid.UUID `json:"job_id"`
	QuizID uuid.UUID `json:"quiz_id"`
	Text   string    `json:"text"`
}

func (w *GenerationRequestedPublisher) processTask(ctx context.Context, t domain.Task) error {
	var msg generationRequestedMsg
	if err := json.Unmarshal(t.Payload, &msg); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("riddler").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.generation_requested_publisher")
	defer span.End()

	slog.InfoContext(ctx, "generation-requested-publisher: публикация", "task_id", t.ID, "job_id", msg.JobID, "quiz_id", msg.QuizID)

	if err := w.publisher.Publish(ctx, natsinf.SubjectGenerationRequested, t.Payload); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
