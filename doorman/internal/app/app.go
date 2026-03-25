package app

import (
	"doorman/internal/config"
	keyhandler "doorman/internal/handler/key"
	otphandler "doorman/internal/handler/otp"
	reghandler "doorman/internal/handler/registration"
	tokenhandler "doorman/internal/handler/token"
	"doorman/internal/infra/db"
	natsinf "doorman/internal/infra/nats"
	"doorman/internal/infra/redis"
	"doorman/internal/repository"
	jwtsvc "doorman/internal/service/jwt"
	otpsvc "doorman/internal/service/otp"
	"doorman/internal/worker"
)

type App struct {
	OtpHandler          *otphandler.Handler
	RegistrationHandler *reghandler.Handler
	TokenHandler        *tokenhandler.Handler
	KeyHandler          *keyhandler.Handler

	OTPRequestConsumer *worker.OTPRequestConsumer
	OTPSentPublisher   *worker.OTPSentPublisher
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

	natsConn, err := natsinf.New(cfg.NATS)
	if err != nil {
		return nil, err
	}

	keyStore, err := repository.NewInMemoryKeysStoreWithOneKey(cfg.Keys)
	if err != nil {
		return nil, err
	}

	otpStore := repository.NewRedisOTPStore(rdb)
	regTokenStore := repository.NewRedisRegTokenStore(rdb)
	refreshTokenStore := repository.NewRedisRefreshTokenStore(rdb)

	identityStore := repository.NewPgIdentityStore(pgdb)
	taskRepo := repository.NewPgTaskRepository(pgdb)

	jwtService := jwtsvc.NewService(keyStore, refreshTokenStore)
	otpService := otpsvc.NewService(identityStore, regTokenStore, otpStore, taskRepo, jwtService)

	tokenHandler := tokenhandler.NewHandler(jwtService)
	otpHandler := otphandler.NewHandler(otpService)
	keyHandler := keyhandler.NewHandler(jwtService)

	natsPublisher := natsinf.NewPublisher(natsConn)
	natsSubscriber := natsinf.NewSubscriber(natsConn)

	return &App{
		OtpHandler:          otpHandler,
		KeyHandler:          keyHandler,
		TokenHandler:        tokenHandler,
		RegistrationHandler: &reghandler.Handler{},

		OTPRequestConsumer: worker.NewOTPRequestConsumer(natsSubscriber, otpService),
		OTPSentPublisher:   worker.NewOTPSentPublisher(taskRepo, natsPublisher),
	}, nil
}
