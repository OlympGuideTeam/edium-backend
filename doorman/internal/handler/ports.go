package handler

import (
	"context"
	"doorman/internal/domain"
	"doorman/internal/transport/dto"
)

type IOTPService interface {
	SendOTP(ctx context.Context, phone string, channel domain.Channel) error
	// VerifyOTP(phone string, otp int64) (bool, error)
}

type IKeyService interface {
	GetPublicKeys() dto.JWKSResponse
}
