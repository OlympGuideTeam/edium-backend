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
	generationCompletedPollInterval = 2 * time.Second
	generationCompletedBatchSize    = 5
	generationCompletedRetryAfter   = 30 * time.Second
)

type generationCompletedTaskRepository interface {
	ClaimPending(ctx context.Context, taskType domain.TaskType, limit int) ([]domain.Task, error)
	MarkDone(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string, retryAfter time.Duration) error
}

type generationAdder interface {
	AddGeneratedQuestions(ctx context.Context, quizID uuid.UUID, questions []domain.AddQuestionParams) error
}

type GenerationCompletedProcessor struct {
	tasks   generationCompletedTaskRepository
	service generationAdder
}

func NewGenerationCompletedProcessor(tasks generationCompletedTaskRepository, service generationAdder) *GenerationCompletedProcessor {
	return &GenerationCompletedProcessor{tasks: tasks, service: service}
}

func (w *GenerationCompletedProcessor) Run(ctx context.Context) error {
	slog.Info("generation-completed-processor: запущен", "interval", generationCompletedPollInterval)
	ticker := time.NewTicker(generationCompletedPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("generation-completed-processor: ошибка батча", "err", err)
			}
		}
	}
}

func (w *GenerationCompletedProcessor) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.TaskTypeGenerationCompleted, generationCompletedBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("generation-completed-processor: ошибка задачи", "task_id", t.ID, "err", err)
			if mfErr := w.tasks.MarkFailed(context.WithoutCancel(ctx), t.ID, err.Error(), generationCompletedRetryAfter); mfErr != nil {
				slog.Error("generation-completed-processor: не удалось сохранить ошибку задачи", "task_id", t.ID, "err", mfErr)
			}
		}
	}
	return nil
}

type sphinxQuestion struct {
	Type     string   `json:"type"`
	Question string   `json:"question"`
	Answer   any      `json:"answer"`
	Options  []string `json:"options"`
}

type sphinxCompletedPayload struct {
	JobID     uuid.UUID        `json:"job_id"`
	QuizID    uuid.UUID        `json:"quiz_id"`
	TraceCtx  string           `json:"trace_ctx"`
	Questions []sphinxQuestion `json:"questions"`
}

func (w *GenerationCompletedProcessor) processTask(ctx context.Context, t domain.Task) error {
	var payload sphinxCompletedPayload
	if err := json.Unmarshal(t.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("riddler").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.generation_completed_processor")
	defer span.End()

	slog.InfoContext(ctx, "generation-completed-processor: обработка", "task_id", t.ID, "quiz_id", payload.QuizID, "questions", len(payload.Questions))

	params, err := mapSphinxQuestions(payload.Questions)
	if err != nil {
		return fmt.Errorf("map sphinx questions: %w", err)
	}

	if len(params) == 0 {
		return w.tasks.MarkDone(ctx, t.ID)
	}

	if err := w.service.AddGeneratedQuestions(ctx, payload.QuizID, params); err != nil {
		return fmt.Errorf("add generated questions: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}

func mapSphinxQuestions(questions []sphinxQuestion) ([]domain.AddQuestionParams, error) {
	params := make([]domain.AddQuestionParams, 0, len(questions))
	for _, q := range questions {
		p, err := mapOneSphinxQuestion(q)
		if err != nil {
			return nil, fmt.Errorf("вопрос %q: %w", q.Question, err)
		}
		params = append(params, p)
	}
	return params, nil
}

func mapOneSphinxQuestion(q sphinxQuestion) (domain.AddQuestionParams, error) {
	switch q.Type {
	case "single_choice":
		answer, ok := q.Answer.(string)
		if !ok {
			return domain.AddQuestionParams{}, fmt.Errorf("single_choice: answer должен быть строкой")
		}
		options := make([]domain.AddOptionParams, len(q.Options))
		for i, opt := range q.Options {
			options[i] = domain.AddOptionParams{Text: opt, IsCorrect: opt == answer}
		}
		return domain.AddQuestionParams{
			Type:     domain.QuestionTypeSingleChoice,
			Text:     q.Question,
			MaxScore: 10,
			Options:  options,
		}, nil

	case "multiple_choice":
		answers, ok := q.Answer.([]any)
		if !ok {
			return domain.AddQuestionParams{}, fmt.Errorf("multiple_choice: answer должен быть массивом")
		}
		correctSet := make(map[string]bool, len(answers))
		for _, a := range answers {
			if s, ok := a.(string); ok {
				correctSet[s] = true
			}
		}
		options := make([]domain.AddOptionParams, len(q.Options))
		for i, opt := range q.Options {
			options[i] = domain.AddOptionParams{Text: opt, IsCorrect: correctSet[opt]}
		}
		return domain.AddQuestionParams{
			Type:     domain.QuestionTypeMultipleChoice,
			Text:     q.Question,
			MaxScore: 10,
			Options:  options,
		}, nil

	case "short_answer":
		return domain.AddQuestionParams{
			Type:     domain.QuestionTypeWithFreeAnswer,
			Text:     q.Question,
			MaxScore: 10,
		}, nil

	default:
		return domain.AddQuestionParams{}, fmt.Errorf("неизвестный тип: %s", q.Type)
	}
}
