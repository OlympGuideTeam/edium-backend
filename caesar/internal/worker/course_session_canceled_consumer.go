package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"caesar/internal/domain"
	natsinf "caesar/internal/infra/nats"
)

type CourseSessionCanceledConsumer struct {
	subscriber *natsinf.Subscriber
	tasks      taskScheduler
}

func NewCourseSessionCanceledConsumer(subscriber *natsinf.Subscriber, tasks taskScheduler) *CourseSessionCanceledConsumer {
	return &CourseSessionCanceledConsumer{subscriber: subscriber, tasks: tasks}
}

func (c *CourseSessionCanceledConsumer) Run(ctx context.Context) error {
	slog.Info("course-session-canceled-consumer: подписка", "subject", natsinf.SubjectCourseSessionCanceled)
	return c.subscriber.QueueSubscribe(ctx, natsinf.SubjectCourseSessionCanceled, natsinf.QueueCourseSessionCanceled, c.handle)
}

type courseSessionCanceledMsg struct {
	SessionID string `json:"session_id"`
}

func (c *CourseSessionCanceledConsumer) handle(ctx context.Context, data []byte) error {
	var msg courseSessionCanceledMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}
	slog.InfoContext(ctx, "course-session-canceled-consumer: получено", "session_id", msg.SessionID)
	return c.tasks.Schedule(ctx, domain.CourseSessionCanceled, data)
}
