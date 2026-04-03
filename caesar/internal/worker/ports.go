package worker

import (
	"caesar/internal/domain"
	"context"
	"time"

	"github.com/google/uuid"
)

type taskScheduler interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
}

type taskRepository interface {
	ClaimPending(ctx context.Context, taskType domain.TaskType, limit int) ([]domain.Task, error)
	MarkDone(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string, retryAfter time.Duration) error
}
