package app

import (
	"doorman/internal/config"
	"doorman/internal/keys"
	"doorman/internal/otp"
	"doorman/internal/registration"
	"doorman/internal/tokens"
)

type App struct {
	OtpHandler          *otp.Handler
	RegistrationHandler *registration.Handler
	TokensHandler       *tokens.Handler
	KeysHandler         *keys.Handler
}

func New(cfg *config.Config) (*App, error) {

	return &App{
		OtpHandler:          &otp.Handler{},
		RegistrationHandler: &registration.Handler{},
		TokensHandler:       &tokens.Handler{},
		KeysHandler:         &keys.Handler{},
	}, nil
}
