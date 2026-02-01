package app

import (
	"doorman/internal/config"
	"doorman/internal/infra/postgres"
	"doorman/internal/infra/redis"
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
	_, err := redis.New(cfg.Redis)
	if err != nil {
		return nil, err
	}

	_, err = postgres.New(cfg.Postgres)
	if err != nil {
		return nil, err
	}

	//producer := kafka.NewProducer(cfg.Kafka)
	//userDeletedConsumer := kafka.NewConsumer(
	//	cfg.Kafka.Brokers,
	//	cfg.Kafka.ClientID,
	//	cfg.Kafka.UserDeletedTopic,
	//)

	return &App{
		OtpHandler:          &otp.Handler{},
		RegistrationHandler: &registration.Handler{},
		TokensHandler:       &tokens.Handler{},
		KeysHandler:         &keys.Handler{},
	}, nil
}
