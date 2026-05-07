package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"herald/internal/domain"
	natsinf "herald/internal/infra/nats"
	"log/slog"

	"github.com/google/uuid"
)

type QuizGenerationNotifyConsumer struct {
	subscriber *natsinf.Subscriber
	tasks      otpSentScheduler
}

func NewQuizGenerationNotifyConsumer(subscriber *natsinf.Subscriber, tasks otpSentScheduler) *QuizGenerationNotifyConsumer {
	return &QuizGenerationNotifyConsumer{subscriber: subscriber, tasks: tasks}
}

func (c *QuizGenerationNotifyConsumer) Run(ctx context.Context) error {
	slog.Info("quiz-generation-notify-consumer: подписка", "subject", natsinf.SubjectQuizGenerationNotify)
	return c.subscriber.QueueSubscribe(ctx, natsinf.SubjectQuizGenerationNotify, natsinf.QueueQuizGenerationNotify, c.handle)
}

type quizGenerationNotifyMsg struct {
	QuizID   uuid.UUID `json:"quiz_id"`
	AuthorID uuid.UUID `json:"author_id"`
	Title    string    `json:"title"`
}

func (c *QuizGenerationNotifyConsumer) handle(ctx context.Context, data []byte) error {
	var msg quizGenerationNotifyMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}
	if msg.AuthorID == uuid.Nil {
		return nil
	}
	payload, _ := json.Marshal(pushNotificationPayload{
		UserID: msg.AuthorID,
		Title:  "Генерация завершена",
		Body:   fmt.Sprintf("Вопросы для квиза «%s» готовы", msg.Title),
		Data: map[string]string{
			"route": "/template/" + msg.QuizID.String(),
			"role":  "teacher",
		},
	})
	return c.tasks.Schedule(ctx, domain.PushNotification, payload)
}
