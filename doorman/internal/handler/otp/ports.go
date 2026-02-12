package otphandler

import (
	"context"
	"doorman/internal/domain"
)

type VerifyResult interface {
	IsVerifyResult()
}

type RegistrationToken struct {
	Token string
}

func (RegistrationToken) IsVerifyResult() {}

type IOTPService interface {
	SendOTP(ctx context.Context, phone string, channel domain.Channel) error
	VerifyOTP(ctx context.Context, phone string, otp uint64) (VerifyResult, error)
}
