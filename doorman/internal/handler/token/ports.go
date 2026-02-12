package tokenhandler

import (
	"context"
)

type AuthTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    uint64
}

func (AuthTokens) IsVerifyResult() {}

type ITokenService interface {
	Refresh(ctx context.Context, refreshToken string) (*AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
}
