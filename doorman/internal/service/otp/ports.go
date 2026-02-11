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
	Exists(ctx context.Context, phone string) (bool, error)
	Get(ctx context.Context, phone string) (*OtpData, error)
	Delete(ctx context.Context, phone string) error
	IncrAttempts(ctx context.Context, phone string) error
}

type RegTokenStore interface {
	Save(ctx context.Context, phone string, regToken string, ttl time.Duration) error
}

type IdentityStore interface {
	Create(ctx context.Context, identity domain.Identity) error
	GetByPhone(ctx context.Context, phone string) (*domain.Identity, error)
}

type KeyStore interface {
	GenerateAuthTokens(userID string, accessTTL time.Duration, refreshTTL time.Duration) (
		accessToken string, refreshToken string, expiresIn int64, err error,
	)
}

type TaskScheduler interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
}
