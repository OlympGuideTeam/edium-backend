package otphandler

import (
	"context"
	"doorman/internal/domain"
)

type VerifyResult interface {
	isVerifyResult()
}

type AuthTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    uint64
}

func (AuthTokens) isVerifyResult() {}

type RegistrationToken struct {
	Token string
}

func (RegistrationToken) isVerifyResult() {}

type IOTPService interface {
	SendOTP(ctx context.Context, phone string, channel domain.Channel) error
	VerifyOTP(ctx context.Context, phone string, otp uint64) (VerifyResult, error)
}
