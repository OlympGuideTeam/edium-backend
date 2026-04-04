package completion

import (
	"charon/internal/domain"
	"context"
	"time"

	"github.com/google/uuid"
)

type LLMClient interface {
	Complete(ctx context.Context, req domain.CompletionRequest) (*domain.CompletionResponse, error)
}

type UsageLogger interface {
	LogUsage(ctx context.Context, record domain.UsageRecord) error
}

type RateLimiter interface {
	Allow(ctx context.Context, service string) (bool, error)
}

type TaskScheduler interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
}

type TaskRepository interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
	ClaimPending(ctx context.Context, taskType domain.TaskType, limit int) ([]domain.Task, error)
	MarkDone(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string, retryAfter time.Duration) error
}
