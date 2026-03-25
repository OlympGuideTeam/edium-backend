package otpsvc

import (
	"context"
	"herald/internal/domain"

	"github.com/google/uuid"
)

type TxManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type TaskRepository interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
	MarkDone(ctx context.Context, id uuid.UUID) error
}

type PendingOTPRepository interface {
	Save(ctx context.Context, correlationID string, chatID int64) error
	Get(ctx context.Context, correlationID string) (*domain.PendingOTP, error)
	Delete(ctx context.Context, correlationID string) error
}
