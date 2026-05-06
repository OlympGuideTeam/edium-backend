package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"

	"riddler/internal/domain"
	natsinf "riddler/internal/infra/nats"
	"riddler/internal/infra/telemetry"
)

const (
	quizGenerationNotifyPollInterval = 2 * time.Second
	quizGenerationNotifyBatchSize    = 10
	quizGenerationNotifyRetryAfter   = 30 * time.Second
)

type QuizGenerationNotifyPublisher struct {
	tasks     taskRepository
	publisher natsPublisher
}

func NewQuizGenerationNotifyPublisher(tasks taskRepository, publisher natsPublisher) *QuizGenerationNotifyPublisher {
	return &QuizGenerationNotifyPublisher{tasks: tasks, publisher: publisher}
}

func (w *QuizGenerationNotifyPublisher) Run(ctx context.Context) error {
	slog.Info("quiz-generation-notify-publisher: запущен", "interval", quizGenerationNotifyPollInterval)
	ticker := time.NewTicker(quizGenerationNotifyPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("quiz-generation-notify-publisher: ошибка батча", "err", err)
			}
		}
	}
}

func (w *QuizGenerationNotifyPublisher) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.TaskTypeQuizGenerationNotify, quizGenerationNotifyBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("quiz-generation-notify-publisher: ошибка задачи", "task_id", t.ID, "err", err)
			_ = w.tasks.MarkFailed(ctx, t.ID, err.Error(), quizGenerationNotifyRetryAfter)
		}
	}
	return nil
}

func (w *QuizGenerationNotifyPublisher) processTask(ctx context.Context, t domain.Task) error {
	ctx, span := otel.Tracer("riddler").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.quiz_generation_notify_publisher")
	defer span.End()

	if err := w.publisher.Publish(ctx, natsinf.SubjectQuizGenerationNotify, t.Payload); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
