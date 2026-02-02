package app

import (
	"doorman/internal/config"
	"doorman/internal/handler"
	"doorman/internal/infra/db"
	"doorman/internal/infra/redis"
	"doorman/internal/repository"
	otpsvc "doorman/internal/service/otp"
)

type App struct {
	OtpHandler          *handler.OTPHandler
	RegistrationHandler *handler.RegistrationHandler
	TokenHandler        *handler.TokenHandler
	KeysHandler         *handler.KeysHandler
}

func New(cfg *config.Config) (*App, error) {
	rdb, err := redis.New(cfg.Redis)
	if err != nil {
		return nil, err
	}

	pgdb, err := db.NewDB(cfg.Postgres)
	if err != nil {
		return nil, err
	}

	//producer := kafka.NewProducer(cfg.Kafka)
	//userDeletedConsumer := kafka.NewConsumer(
	//	cfg.Kafka.Brokers,
	//	cfg.Kafka.ClientID,
	//	cfg.Kafka.UserDeletedTopic,
	//)

	txManager := db.NewTxManager(pgdb)

	otpStore := repository.NewRedisOTPStore(rdb)
	identityStore := repository.NewPgIdentityStore(pgdb)
	scheduler := repository.NewPgScheduler(pgdb)

	otpService := otpsvc.NewService(txManager, identityStore, scheduler, otpStore)

	otpHandler := handler.NewHandler(otpService)

	return &App{
		OtpHandler:          otpHandler,
		TokenHandler:        &handler.TokenHandler{},
		KeysHandler:         &handler.KeysHandler{},
		RegistrationHandler: &handler.RegistrationHandler{},
	}, nil
}
