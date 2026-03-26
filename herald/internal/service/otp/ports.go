package otpsvc

import (
	"context"
	"herald/internal/domain"
)

type TxManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type TaskRepository interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
}

type PendingOTPRepository interface {
	Save(ctx context.Context, phone string, chatID int64) error
	Get(ctx context.Context, phone string) (*domain.PendingOTP, error)
	Delete(ctx context.Context, phone string) error
}
