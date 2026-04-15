package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"caesar/internal/domain"
	natsinf "caesar/internal/infra/nats"
)

type AttemptScoredConsumer struct {
	subscriber *natsinf.Subscriber
	tasks      taskScheduler
}

func NewAttemptScoredConsumer(subscriber *natsinf.Subscriber, tasks taskScheduler) *AttemptScoredConsumer {
	return &AttemptScoredConsumer{subscriber: subscriber, tasks: tasks}
}

func (c *AttemptScoredConsumer) Run(ctx context.Context) error {
	slog.Info("attempt-scored-consumer: подписка", "subject", natsinf.SubjectAttemptScored)
	return c.subscriber.QueueSubscribe(ctx, natsinf.SubjectAttemptScored, natsinf.QueueAttemptScored, c.handle)
}

type attemptScoredMsg struct {
	AttemptID  string  `json:"attempt_id"`
	SessionID  string  `json:"session_id"`
	UserID     string  `json:"user_id"`
	TotalScore float64 `json:"total_score"`
}

func (c *AttemptScoredConsumer) handle(ctx context.Context, data []byte) error {
	var msg attemptScoredMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}
	slog.InfoContext(ctx, "attempt-scored-consumer: получено", "attempt_id", msg.AttemptID)
	return c.tasks.Schedule(ctx, domain.AttemptScored, data)
}
