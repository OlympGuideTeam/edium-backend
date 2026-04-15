package worker

import (
	"context"
	"time"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

type taskRepository interface {
	ClaimPending(ctx context.Context, taskType domain.TaskType, limit int) ([]domain.Task, error)
	MarkDone(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string, retryAfter time.Duration) error
}

type natsPublisher interface {
	Publish(ctx context.Context, subject string, data []byte) error
}
