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
	quizTemplateAttachedPollInterval = 2 * time.Second
	quizTemplateAttachedBatchSize    = 10
	quizTemplateAttachedRetryAfter   = 30 * time.Second
)

type QuizTemplateAttachedPublisher struct {
	tasks     taskRepository
	publisher natsPublisher
}

func NewQuizTemplateAttachedPublisher(tasks taskRepository, publisher natsPublisher) *QuizTemplateAttachedPublisher {
	return &QuizTemplateAttachedPublisher{tasks: tasks, publisher: publisher}
}

func (w *QuizTemplateAttachedPublisher) Run(ctx context.Context) error {
	slog.Info("quiz-template-attached-publisher: запущен")
	ticker := time.NewTicker(quizTemplateAttachedPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("quiz-template-attached-publisher: ошибка батча", "err", err)
			}
		}
	}
}

type quizTemplateAttachedMsg struct {
	QuizTemplateID string `json:"quiz_template_id"`
	ModuleID       string `json:"module_id"`
}

func (w *QuizTemplateAttachedPublisher) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.TaskTypeQuizTemplateAttachedPublisher, quizTemplateAttachedBatchSize)
	if err != nil {
		return fmt.Errorf("claim pending: %w", err)
	}
	for i := range tasks {
		if err := w.processTask(ctx, tasks[i]); err != nil {
			slog.Error("quiz-template-attached-publisher: ошибка задачи", "task_id", tasks[i].ID, "err", err)
			_ = w.tasks.MarkFailed(ctx, tasks[i].ID, err.Error(), quizTemplateAttachedRetryAfter)
		}
	}
	return nil
}

func (w *QuizTemplateAttachedPublisher) processTask(ctx context.Context, t domain.Task) error {
	var msg quizTemplateAttachedMsg
	if err := json.Unmarshal(t.Payload, &msg); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("riddler").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.quiz_template_attached_publisher")
	defer span.End()

	if err := w.publisher.Publish(ctx, natsinf.SubjectQuizTemplateAttached, t.Payload); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
