package app

import (
	"context"
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
	regsvc "doorman/internal/service/registration"
	"doorman/internal/worker"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type App struct {
	OtpHandler          *otphandler.Handler
	RegistrationHandler *reghandler.Handler
	TokenHandler        *tokenhandler.Handler
	KeyHandler          *keyhandler.Handler

	OTPRequestConsumer   *worker.OTPRequestConsumer
	OTPRequestProcessor  *worker.OTPRequestProcessor
	OTPSentPublisher     *worker.OTPSentPublisher
	UserCreatedPublisher *worker.UserCreatedPublisher
	UserDeletedConsumer  *worker.UserDeletedConsumer
	UserDeletedProcessor *worker.UserDeletedProcessor
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

	txManager := db.NewTxManager(pgdb)

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
	registrationService := regsvc.NewService(identityStore, regTokenStore, taskRepo, jwtService, txManager)

	tokenHandler := tokenhandler.NewHandler(jwtService)
	otpHandler := otphandler.NewHandler(otpService)
	keyHandler := keyhandler.NewHandler(jwtService)
	registrationHandler := reghandler.NewHandler(registrationService)

	natsPublisher := natsinf.NewPublisher(natsConn)
	natsSubscriber := natsinf.NewSubscriber(natsConn)

	a := &App{
		OtpHandler:          otpHandler,
		KeyHandler:          keyHandler,
		TokenHandler:        tokenHandler,
		RegistrationHandler: registrationHandler,

		OTPRequestConsumer:   worker.NewOTPRequestConsumer(natsSubscriber, taskRepo),
		OTPRequestProcessor:  worker.NewOTPRequestProcessor(taskRepo, otpService),
		OTPSentPublisher:     worker.NewOTPSentPublisher(taskRepo, natsPublisher),
		UserCreatedPublisher: worker.NewUserCreatedPublisher(taskRepo, natsPublisher),
		UserDeletedConsumer:  worker.NewUserDeletedConsumer(natsSubscriber, taskRepo),
		UserDeletedProcessor: worker.NewUserDeletedProcessor(taskRepo, identityStore, refreshTokenStore),
	}
	return a, nil
}

func (a *App) Workers() map[string]func(context.Context) error {
	return map[string]func(context.Context) error{
		"OTPRequestConsumer":   a.OTPRequestConsumer.Run,
		"OTPRequestProcessor":  a.OTPRequestProcessor.Run,
		"OTPSentPublisher":     a.OTPSentPublisher.Run,
		"UserCreatedPublisher": a.UserCreatedPublisher.Run,
		"UserDeletedConsumer":  a.UserDeletedConsumer.Run,
		"UserDeletedProcessor": a.UserDeletedProcessor.Run,
	}
}

func (a *App) Router(serviceName string) *gin.Engine {
	r := gin.Default()
	r.Use(otelgin.Middleware(serviceName))

	api := r.Group("/doorman/v1")
	api.POST("/otp/send", a.OtpHandler.Send)
	api.POST("/otp/verify", a.OtpHandler.Verify)
	api.POST("/auth/register", a.RegistrationHandler.Register)
	api.POST("/auth/refresh", a.TokenHandler.Refresh)
	api.POST("/auth/logout", a.TokenHandler.Logout)
	api.GET("/.well-known/jwks.json", a.KeyHandler.GetJWKS)

	return r
}

func (a *App) Run(ctx context.Context, cfg *config.Config) error {
	for name, run := range a.Workers() {
		go func() {
			if err := run(ctx); err != nil {
				slog.Error("воркер завершился с ошибкой", "worker", name, "err", err)
			}
		}()
	}
	if err := a.Router(cfg.OTel.ServiceName).Run(fmt.Sprintf(":%d", cfg.App.Port)); err != nil {
		return fmt.Errorf("сервер: %w", err)
	}
	return nil
}
