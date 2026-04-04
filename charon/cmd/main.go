package main

import (
	"charon/internal/app"
	"charon/internal/config"
	"charon/internal/infra/telemetry"
	"charon/internal/pkg/logger"
	"context"
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
		slog.Error("config load error", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg); err != nil {
		slog.Error("application stopped with error", "err", err)
	}
}

func run(ctx context.Context, cfg *config.Config) error {
	shutdown, err := telemetry.Init(ctx, cfg.OTel.Endpoint, cfg.OTel.ServiceName)
	if err != nil {
		return err
	}
	defer func() { _ = shutdown(context.Background()) }()

	application, err := app.New(cfg)
	if err != nil {
		return err
	}

	return application.Run(ctx, cfg)
}
