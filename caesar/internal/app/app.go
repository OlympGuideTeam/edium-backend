package app

import (
	"context"
	"fmt"
	"log/slog"

	"caesar/internal/config"
	classhandler "caesar/internal/handler/class"
	userhandler "caesar/internal/handler/user"
	"caesar/internal/infra/db"
	"caesar/internal/infra/jwks"
	natsinf "caesar/internal/infra/nats"
	"caesar/internal/middleware"
	"caesar/internal/repository"
	classsvc "caesar/internal/service/class"
	usersvc "caesar/internal/service/user"
	"caesar/internal/worker"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type App struct {
	UserHandler  *userhandler.Handler
	ClassHandler *classhandler.Handler

	UserCreatedConsumer  *worker.UserCreatedConsumer
	UserCreatedProcessor *worker.UserCreatedProcessor
	UserDeletedPublisher *worker.UserDeletedPublisher

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
	txManager := db.NewTxManager(pgdb)
	publisher := natsinf.NewPublisher(natsConn)

	classStore := repository.NewPgClassStore(pgdb)

	userService := usersvc.NewService(userStore, taskRepo, txManager)
	userHandler := userhandler.NewHandler(userService)

	classService := classsvc.NewService(classStore)
	classHandler := classhandler.NewHandler(classService)

	natsSubscriber := natsinf.NewSubscriber(natsConn)

	return &App{
		UserHandler:          userHandler,
		ClassHandler:         classHandler,
		UserCreatedConsumer:  worker.NewUserCreatedConsumer(natsSubscriber, taskRepo),
		UserCreatedProcessor: worker.NewUserCreatedProcessor(taskRepo, userStore),
		UserDeletedPublisher: worker.NewUserDeletedPublisher(taskRepo, publisher),
		jwksClient:           jwksClient,
	}, nil
}

func (a *App) Workers() map[string]func(context.Context) error {
	return map[string]func(context.Context) error{
		"UserCreatedConsumer":  a.UserCreatedConsumer.Run,
		"UserCreatedProcessor": a.UserCreatedProcessor.Run,
		"UserDeletedPublisher": a.UserDeletedPublisher.Run,
	}
}

func (a *App) Router(serviceName string) *gin.Engine {
	r := gin.Default()
	r.Use(otelgin.Middleware(serviceName))

	auth := middleware.Auth(a.jwksClient)

	api := r.Group("/caesar/v1")
	api.Use(auth)
	api.GET("/users/me", a.UserHandler.GetMe)
	api.PATCH("/users/me", a.UserHandler.UpdateMe)
	api.DELETE("/users/me", a.UserHandler.DeleteMe)

	api.GET("/classes/me", a.ClassHandler.GetMyClasses)
	api.POST("/classes", a.ClassHandler.CreateClass)
	api.GET("/classes/:classId", a.ClassHandler.GetClass)
	api.PATCH("/classes/:classId", a.ClassHandler.UpdateClass)
	api.DELETE("/classes/:classId", a.ClassHandler.DeleteClass)
	api.DELETE("/classes/:classId/members/:userId", a.ClassHandler.RemoveMember)
	api.GET("/classes/:classId/invite", a.ClassHandler.GetInviteLink)
	api.POST("/invitations/:invitationId/accept", a.ClassHandler.AcceptInvitation)

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
