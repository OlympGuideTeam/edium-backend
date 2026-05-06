package main

import (
	"context"
	"herald/internal/app"
	"herald/internal/config"
	"herald/internal/infra/telemetry"
	"herald/internal/pkg/logger"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	slog.SetDefault(slog.New(logger.NewContextHandler(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
	)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("загрузка конфигурации", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg); err != nil {
		slog.Error("приложение завершилось с ошибкой", "err", err)
	}
}

func run(ctx context.Context, cfg *config.Config) error {
	shutdown, err := telemetry.Init(ctx, cfg.OTel.Endpoint, cfg.OTel.ServiceName)
	if err != nil {
		return err
	}
	defer func() { _ = shutdown(context.Background()) }()

	application, err := app.New(ctx, cfg)
	if err != nil {
		return err
	}

	return application.Run(ctx)
}
