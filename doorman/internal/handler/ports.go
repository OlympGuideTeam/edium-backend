package handler

import (
	"context"
	"doorman/internal/domain"
)

type IOTPService interface {
	SendOTP(ctx context.Context, phone string, channel domain.Channel) error
	// VerifyOTP(phone string, otp int64) (bool, error)
}
