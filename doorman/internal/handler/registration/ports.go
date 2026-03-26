package reghandler

import (
	"context"
	tokenhandler "doorman/internal/handler/token"
)

type IRegistrationService interface {
	Register(ctx context.Context, phone, name, surname, regToken string) (*tokenhandler.AuthTokens, error)
}
