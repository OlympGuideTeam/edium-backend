package worker

import (
	"context"
	"fmt"
	"log/slog"

	"riddler/internal/domain"
	natsinf "riddler/internal/infra/nats"
)

type courseSessionDeletedScheduler interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
}

type CourseSessionDeletedConsumer struct {
	subscriber *natsinf.Subscriber
	tasks      courseSessionDeletedScheduler
}

func NewCourseSessionDeletedConsumer(subscriber *natsinf.Subscriber, tasks courseSessionDeletedScheduler) *CourseSessionDeletedConsumer {
	return &CourseSessionDeletedConsumer{subscriber: subscriber, tasks: tasks}
}

func (c *CourseSessionDeletedConsumer) Run(ctx context.Context) error {
	slog.Info("course-session-deleted-consumer: подписка", "subject", natsinf.SubjectCourseSessionDeleted)
	return c.subscriber.QueueSubscribe(ctx, natsinf.SubjectCourseSessionDeleted, "riddler-course-session-deleted", c.handle)
}

func (c *CourseSessionDeletedConsumer) handle(ctx context.Context, data []byte) error {
	slog.InfoContext(ctx, "course-session-deleted-consumer: получено")
	if err := c.tasks.Schedule(ctx, domain.TaskTypeCourseSessionDeleted, data); err != nil {
		return fmt.Errorf("schedule: %w", err)
	}
	return nil
}
