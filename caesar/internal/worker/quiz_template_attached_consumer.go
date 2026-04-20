package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"caesar/internal/domain"
	natsinf "caesar/internal/infra/nats"
)

type QuizTemplateAttachedConsumer struct {
	subscriber *natsinf.Subscriber
	tasks      taskScheduler
}

func NewQuizTemplateAttachedConsumer(subscriber *natsinf.Subscriber, tasks taskScheduler) *QuizTemplateAttachedConsumer {
	return &QuizTemplateAttachedConsumer{subscriber: subscriber, tasks: tasks}
}

func (c *QuizTemplateAttachedConsumer) Run(ctx context.Context) error {
	slog.Info("quiz-template-attached-consumer: подписка", "subject", natsinf.SubjectQuizTemplateAttached)
	return c.subscriber.QueueSubscribe(ctx, natsinf.SubjectQuizTemplateAttached, natsinf.QueueQuizTemplateAttached, c.handle)
}

type quizTemplateAttachedMsg struct {
	QuizTemplateID string `json:"quiz_template_id"`
	CourseID       string `json:"course_id"`
	Title          string `json:"title"`
}

func (c *QuizTemplateAttachedConsumer) handle(ctx context.Context, data []byte) error {
	var msg quizTemplateAttachedMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}
	slog.InfoContext(ctx, "quiz-template-attached-consumer: получено", "quiz_template_id", msg.QuizTemplateID)
	return c.tasks.Schedule(ctx, domain.QuizTemplateAttached, data)
}
