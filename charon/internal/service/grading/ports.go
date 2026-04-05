package grading

import (
	"charon/internal/domain"
	"context"
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
