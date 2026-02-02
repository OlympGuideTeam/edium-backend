package otpsvc

import (
	"context"
	"doorman/internal/domain"
	"time"
)

type OTPStore interface {
	Save(
		ctx context.Context,
		phone string,
		hashOtp string,
		ttl time.Duration,
	) error
	Exists(ctx context.Context, phone string) (exist bool, err error)
	// Get(ctx context.Context, phone string) (otp int64, err error)
	// Delete(ctx context.Context, phone string) error
	// incr, decr?
}

type IdentityStore interface {
	Create(ctx context.Context, identity domain.Identity) error
	GetByPhone(ctx context.Context, phone string) (*domain.Identity, error)
}

type TaskScheduler interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
}
