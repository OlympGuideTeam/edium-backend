package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"herald/internal/domain"
	natsinf "herald/internal/infra/nats"
	"log/slog"
	"math"

	"github.com/google/uuid"
)

type AttemptScoredConsumer struct {
	subscriber *natsinf.Subscriber
	tasks      otpSentScheduler
}

func NewAttemptScoredConsumer(subscriber *natsinf.Subscriber, tasks otpSentScheduler) *AttemptScoredConsumer {
	return &AttemptScoredConsumer{subscriber: subscriber, tasks: tasks}
}

func (c *AttemptScoredConsumer) Run(ctx context.Context) error {
	slog.Info("attempt-scored-consumer: подписка", "subject", natsinf.SubjectAttemptScored)
	return c.subscriber.QueueSubscribe(ctx, natsinf.SubjectAttemptScored, natsinf.QueueAttemptScored, c.handle)
}

type attemptScoredMsg struct {
	AttemptID  uuid.UUID `json:"attempt_id"`
	SessionID  uuid.UUID `json:"session_id"`
	UserID     uuid.UUID `json:"user_id"`
	AuthorID   uuid.UUID `json:"author_id"`
	TotalScore float64   `json:"total_score"`
	MaxScore   float64   `json:"max_score"`
	GradedBy   string    `json:"graded_by"`
}

func (c *AttemptScoredConsumer) handle(ctx context.Context, data []byte) error {
	var msg attemptScoredMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}

	if msg.GradedBy == "llm" {
		if msg.AuthorID == uuid.Nil {
			return nil
		}
		payload, _ := json.Marshal(pushNotificationPayload{
			UserID: msg.AuthorID,
			Title:  "AI завершил проверку",
			Body:   "Работы учеников проверены AI — ознакомьтесь с результатами",
			Data: map[string]string{
				"route": "/test/" + msg.SessionID.String() + "/results",
				"role":  "teacher",
			},
		})
		return c.tasks.Schedule(ctx, domain.PushNotification, payload)
	}

	if msg.UserID == uuid.Nil {
		return nil
	}
	score := int(math.Round(msg.TotalScore))
	maxScore := int(math.Round(msg.MaxScore))
	body := fmt.Sprintf("Ваш результат: %d / %d", score, maxScore)
	payload, _ := json.Marshal(pushNotificationPayload{
		UserID: msg.UserID,
		Title:  "Результаты готовы",
		Body:   body,
		Data: map[string]string{
			"route": "/test/" + msg.AttemptID.String() + "/review",
			"role":  "student",
		},
	})
	return c.tasks.Schedule(ctx, domain.PushNotification, payload)
}
