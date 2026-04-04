package app

import (
	"charon/internal/client/deepseek"
	"charon/internal/config"
	healthhandler "charon/internal/handler/health"
	"charon/internal/infra/clickhouse"
	"charon/internal/infra/db"
	natsinf "charon/internal/infra/nats"
	"charon/internal/infra/redis"
	"charon/internal/repository"
	completionsvc "charon/internal/service/completion"
	"charon/internal/service/ratelimit"
	"charon/internal/worker"
	"context"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type App struct {
	HealthHandler *healthhandler.Handler

	CompletionConsumer  *worker.CompletionConsumer
	CompletionProcessor *worker.CompletionProcessor
	CompletionPublisher *worker.CompletionPublisher
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

	chConn, err := clickhouse.New(cfg.ClickHouse)
	if err != nil {
		return nil, err
	}

	taskRepo := repository.NewPgTaskRepository(pgdb)
	usageWriter := repository.NewClickHouseUsageWriter(chConn)

	dsClient := deepseek.NewClient(cfg.DeepSeek.BaseURL, cfg.DeepSeek.APIKey, cfg.DeepSeek.Timeout)

	rateLimits := map[string]int{
		"riddler": cfg.RateLimit.Riddler,
		"yoda":    cfg.RateLimit.Yoda,
	}
	limiter := ratelimit.NewLimiter(rdb, rateLimits)

	completionService := completionsvc.NewService(dsClient, usageWriter, limiter, taskRepo)

	natsPublisher := natsinf.NewPublisher(natsConn)
	natsSubscriber := natsinf.NewSubscriber(natsConn)

	a := &App{
		HealthHandler: healthhandler.NewHandler(),

		CompletionConsumer:  worker.NewCompletionConsumer(natsSubscriber, taskRepo),
		CompletionProcessor: worker.NewCompletionProcessor(taskRepo, completionService),
		CompletionPublisher: worker.NewCompletionPublisher(taskRepo, natsPublisher),
	}
	return a, nil
}

func (a *App) Workers() map[string]func(context.Context) error {
	return map[string]func(context.Context) error{
		"CompletionConsumer":  a.CompletionConsumer.Run,
		"CompletionProcessor": a.CompletionProcessor.Run,
		"CompletionPublisher": a.CompletionPublisher.Run,
	}
}

func (a *App) Router(serviceName string) *gin.Engine {
	r := gin.Default()
	r.Use(otelgin.Middleware(serviceName))

	r.GET("/health", a.HealthHandler.Health)

	return r
}

func (a *App) Run(ctx context.Context, cfg *config.Config) error {
	for name, run := range a.Workers() {
		go func() {
			if err := run(ctx); err != nil {
				slog.Error("worker stopped with error", "worker", name, "err", err)
			}
		}()
	}
	if err := a.Router(cfg.OTel.ServiceName).Run(fmt.Sprintf(":%d", cfg.App.Port)); err != nil {
		return fmt.Errorf("server: %w", err)
	}
	return nil
}
