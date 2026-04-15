package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"caesar/internal/domain"
	natsinf "caesar/internal/infra/nats"
)

type CourseSessionCreatedConsumer struct {
	subscriber *natsinf.Subscriber
	tasks      taskScheduler
}

func NewCourseSessionCreatedConsumer(subscriber *natsinf.Subscriber, tasks taskScheduler) *CourseSessionCreatedConsumer {
	return &CourseSessionCreatedConsumer{subscriber: subscriber, tasks: tasks}
}

func (c *CourseSessionCreatedConsumer) Run(ctx context.Context) error {
	slog.Info("course-session-created-consumer: подписка", "subject", natsinf.SubjectCourseSessionCreated)
	return c.subscriber.QueueSubscribe(ctx, natsinf.SubjectCourseSessionCreated, natsinf.QueueCourseSessionCreated, c.handle)
}

type courseSessionCreatedMsg struct {
	SessionID string `json:"session_id"`
	ModuleID  string `json:"module_id"`
}

func (c *CourseSessionCreatedConsumer) handle(ctx context.Context, data []byte) error {
	var msg courseSessionCreatedMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}
	slog.InfoContext(ctx, "course-session-created-consumer: получено", "session_id", msg.SessionID)
	return c.tasks.Schedule(ctx, domain.CourseSessionCreated, data)
}
