package app

import (
	"doorman/internal/config"
	"doorman/internal/handler"
	"doorman/internal/infra/db"
	"doorman/internal/infra/redis"
	"doorman/internal/repository"
	jwtsvc "doorman/internal/service/jwt"
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

	txManager := db.NewTxManager(pgdb)

	keyStore, err := repository.NewInMemoryKeysStoreWithOneKey(cfg.Keys)
	if err != nil {
		return nil, err
	}
	otpStore := repository.NewRedisOTPStore(rdb)
	identityStore := repository.NewPgIdentityStore(pgdb)
	scheduler := repository.NewPgScheduler(pgdb)

	otpService := otpsvc.NewService(txManager, identityStore, scheduler, otpStore)
	jwtService := jwtsvc.NewService(keyStore)

	otpHandler := handler.NewOTPHandler(otpService)
	keysHandler := handler.NewKeysHandler(jwtService)

	return &App{
		OtpHandler:          otpHandler,
		KeysHandler:         keysHandler,
		TokenHandler:        &handler.TokenHandler{},
		RegistrationHandler: &handler.RegistrationHandler{},
	}, nil
}
