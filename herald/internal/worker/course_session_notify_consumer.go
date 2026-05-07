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

type CourseSessionNotifyConsumer struct {
	subscriber *natsinf.Subscriber
	tasks      otpSentScheduler
}

func NewCourseSessionNotifyConsumer(subscriber *natsinf.Subscriber, tasks otpSentScheduler) *CourseSessionNotifyConsumer {
	return &CourseSessionNotifyConsumer{subscriber: subscriber, tasks: tasks}
}

func (c *CourseSessionNotifyConsumer) Run(ctx context.Context) error {
	slog.Info("course-session-notify-consumer: подписка", "subject", natsinf.SubjectCourseSessionNotify)
	return c.subscriber.QueueSubscribe(ctx, natsinf.SubjectCourseSessionNotify, natsinf.QueueCourseSessionNotify, c.handle)
}

type courseSessionNotifyMsg struct {
	SessionID string      `json:"session_id"`
	UserIDs   []uuid.UUID `json:"user_ids"`
	Title     string      `json:"title"`
	Mode      string      `json:"mode"`
}

func (c *CourseSessionNotifyConsumer) handle(ctx context.Context, data []byte) error {
	var msg courseSessionNotifyMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}
	for _, userID := range msg.UserIDs {
		payload, _ := json.Marshal(pushNotificationPayload{
			UserID: userID,
			Title:  "Новый тест назначен",
			Body:   msg.Title,
			Data: map[string]string{
				"route": "/test/" + msg.SessionID,
				"role":  "student",
			},
		})
		if err := c.tasks.Schedule(ctx, domain.PushNotification, payload); err != nil {
			slog.ErrorContext(ctx, "course-session-notify-consumer: не удалось запланировать уведомление",
				"user_id", userID, "session_id", msg.SessionID, "err", err)
		}
	}
	return nil
}
