package app

import (
	"doorman/internal/config"
	"doorman/internal/handler"
	keyhandler "doorman/internal/handler/key"
	otphandler "doorman/internal/handler/otp"
	"doorman/internal/infra/db"
	"doorman/internal/infra/redis"
	"doorman/internal/repository"
	jwtsvc "doorman/internal/service/jwt"
	otpsvc "doorman/internal/service/otp"
)

type App struct {
	OtpHandler          *otphandler.Handler
	RegistrationHandler *handler.RegistrationHandler
	TokenHandler        *handler.TokenHandler
	KeyHandler          *keyhandler.Handler
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

	// txManager := db.NewTxManager(pgdb)

	keyStore, err := repository.NewInMemoryKeysStoreWithOneKey(cfg.Keys)
	if err != nil {
		return nil, err
	}
	otpStore := repository.NewRedisOTPStore(rdb)
	regTokenStore := repository.NewRedisRegTokenStore(rdb)
	identityStore := repository.NewPgIdentityStore(pgdb)
	scheduler := repository.NewPgScheduler(pgdb)

	otpService := otpsvc.NewService(identityStore, regTokenStore, keyStore, otpStore, scheduler)
	jwtService := jwtsvc.NewService(keyStore)

	otpHandler := otphandler.NewHandler(otpService)
	keyHandler := keyhandler.NewHandler(jwtService)

	return &App{
		OtpHandler:          otpHandler,
		KeyHandler:          keyHandler,
		TokenHandler:        &handler.TokenHandler{},
		RegistrationHandler: &handler.RegistrationHandler{},
	}, nil
}
