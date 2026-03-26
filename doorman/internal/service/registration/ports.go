package regsvc

import (
	"context"
	"doorman/internal/domain"
)

type IdentityStore interface {
	Create(ctx context.Context, phone string) (domain.Identity, error)
}

type RegTokenStore interface {
	Get(ctx context.Context, phone string) (string, error)
	Delete(ctx context.Context, phone string) error
}

type TaskScheduler interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
}

type JWTIssuer interface {
	IssueTokens(ctx context.Context, userID string) (string, string, int64, error)
}

type TxManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
