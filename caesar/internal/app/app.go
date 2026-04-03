package app

import (
	"caesar/internal/config"
	userhandler "caesar/internal/handler/user"
	"caesar/internal/infra/db"
	"caesar/internal/infra/jwks"
	natsinf "caesar/internal/infra/nats"
	"caesar/internal/middleware"
	"caesar/internal/repository"
	usersvc "caesar/internal/service/user"
	"caesar/internal/worker"
	"context"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type App struct {
	UserHandler *userhandler.Handler

	UserCreatedConsumer  *worker.UserCreatedConsumer
	UserCreatedProcessor *worker.UserCreatedProcessor

	jwksClient *jwks.Client
}

func New(ctx context.Context, cfg *config.Config) (*App, error) {
	pgdb, err := db.NewDB(cfg.Postgres)
	if err != nil {
		return nil, err
	}

	natsConn, err := natsinf.New(cfg.NATS)
	if err != nil {
		return nil, err
	}

	jwksClient := jwks.NewClient(cfg.Doorman.JWKSEndpoint)
	if err := jwksClient.Load(ctx); err != nil {
		return nil, fmt.Errorf("загрузка JWKS: %w", err)
	}
	jwksClient.StartRefresh(ctx)

	userStore := repository.NewPgUserStore(pgdb)
	taskRepo := repository.NewPgTaskRepository(pgdb)

	userService := usersvc.NewService(userStore)
	userHandler := userhandler.NewHandler(userService)

	natsSubscriber := natsinf.NewSubscriber(natsConn)

	return &App{
		UserHandler:          userHandler,
		UserCreatedConsumer:  worker.NewUserCreatedConsumer(natsSubscriber, taskRepo),
		UserCreatedProcessor: worker.NewUserCreatedProcessor(taskRepo, userStore),
		jwksClient:           jwksClient,
	}, nil
}

func (a *App) Workers() map[string]func(context.Context) error {
	return map[string]func(context.Context) error{
		"UserCreatedConsumer":  a.UserCreatedConsumer.Run,
		"UserCreatedProcessor": a.UserCreatedProcessor.Run,
	}
}

func (a *App) Router(serviceName string) *gin.Engine {
	r := gin.Default()
	r.Use(otelgin.Middleware(serviceName))

	auth := middleware.Auth(a.jwksClient)

	api := r.Group("/caesar/v1")
	api.Use(auth)
	api.GET("/users/me", a.UserHandler.GetMe)

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
