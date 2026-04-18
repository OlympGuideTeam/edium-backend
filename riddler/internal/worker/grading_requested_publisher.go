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
	gradingRequestedPollInterval = 2 * time.Second
	gradingRequestedBatchSize    = 10
	gradingRequestedRetryAfter   = 30 * time.Second
)

type GradingRequestedPublisher struct {
	tasks     taskRepository
	publisher natsPublisher
}

func NewGradingRequestedPublisher(tasks taskRepository, publisher natsPublisher) *GradingRequestedPublisher {
	return &GradingRequestedPublisher{tasks: tasks, publisher: publisher}
}

func (w *GradingRequestedPublisher) Run(ctx context.Context) error {
	slog.Info("grading-requested-publisher: запущен")
	ticker := time.NewTicker(gradingRequestedPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("grading-requested-publisher: ошибка батча", "err", err)
			}
		}
	}
}

func (w *GradingRequestedPublisher) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.TaskTypeGradingRequestedPublisher, gradingRequestedBatchSize)
	if err != nil {
		return fmt.Errorf("claim pending: %w", err)
	}
	for i := range tasks {
		if err := w.processTask(ctx, tasks[i]); err != nil {
			slog.Error("grading-requested-publisher: ошибка задачи", "task_id", tasks[i].ID, "err", err)
			_ = w.tasks.MarkFailed(ctx, tasks[i].ID, err.Error(), gradingRequestedRetryAfter)
		}
	}
	return nil
}

type charonGradeRequest struct {
	RequestID string                `json:"request_id"`
	Question  string                `json:"question"`
	Answers   []charonStudentAnswer `json:"answers"`
}

type charonStudentAnswer struct {
	StudentID string `json:"student_id"`
	Text      string `json:"text"`
}

type incomingGradingPayload struct {
	QuestionText string `json:"question_text"`
	Answers      []struct {
		EvalID string `json:"eval_id"`
		Text   string `json:"text"`
	} `json:"answers"`
}

func (w *GradingRequestedPublisher) processTask(ctx context.Context, t domain.Task) error {
	var p incomingGradingPayload
	if err := json.Unmarshal(t.Payload, &p); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("riddler").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.grading_requested_publisher")
	defer span.End()

	answers := make([]charonStudentAnswer, len(p.Answers))
	for i, a := range p.Answers {
		answers[i] = charonStudentAnswer{StudentID: a.EvalID, Text: a.Text}
	}

	req := charonGradeRequest{
		RequestID: uuid.New().String(),
		Question:  p.QuestionText,
		Answers:   answers,
	}
	data, _ := json.Marshal(req)

	if err := w.publisher.Publish(ctx, natsinf.SubjectQuizGradeRequested, data); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
