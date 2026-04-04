package user

import (
	"caesar/internal/domain"
	"context"

	"github.com/google/uuid"
)

type userStore interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	Update(ctx context.Context, id uuid.UUID, name, surname string) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.UserStatus) error
}

type taskScheduler interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
}

type txRunner interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
