package otp

import (
	"context"
	"doorman/internal/domain"
)

type IOTPService interface {
	SendOTP(context context.Context, phone string, channel channel) error
	// VerifyOTP(phone string, otp int64) (bool, error)
}

type IOTPStore interface {
	Save(ctx context.Context, phone string, otp int64) error
}

type IIdentityStore interface {
	Create(ctx context.Context, identity *domain.Identity) error
	GetByPhone(ctx context.Context, phone string) (*domain.Identity, error)
}

type ITaskStore interface {
	EnqueueOTP(ctx context.Context, phone string, otp int64, channel channel) error
}
