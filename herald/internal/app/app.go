package app

import (
	"context"
	"fmt"
	tgbot "herald/internal/bot/telegram"
	"herald/internal/config"
	"herald/internal/infra/db"
	natsinf "herald/internal/infra/nats"
	tginfra "herald/internal/infra/telegram"
	"herald/internal/repository"
	otpsvc "herald/internal/service/otp"
	"herald/internal/worker"
	"log/slog"
)

type App struct {
	TGHandler           *tgbot.Handler
	OTPRequestPublisher *worker.OTPRequestPublisher
	OTPSentConsumer     *worker.OTPSentConsumer
	OTPSentProcessor    *worker.OTPSentProcessor
}

func New(cfg *config.Config) (*App, error) {
	pgdb, err := db.NewDB(cfg.Postgres)
	if err != nil {
		return nil, err
	}

	natsConn, err := natsinf.New(cfg.NATS)
	if err != nil {
		return nil, err
	}

	bot, err := tginfra.New(cfg.Telegram)
	if err != nil {
		return nil, err
	}

	txManager := db.NewTxManager(pgdb)
	taskRepo := repository.NewPgTaskRepository(pgdb)
	pendingOTPRepo := repository.NewPgPendingOTPRepository(pgdb)

	otpService := otpsvc.NewService(txManager, taskRepo, pendingOTPRepo)

	natsPublisher := natsinf.NewPublisher(natsConn)
	natsSubscriber := natsinf.NewSubscriber(natsConn)

	return &App{
		TGHandler:           tgbot.NewHandler(bot, otpService),
		OTPRequestPublisher: worker.NewOTPRequestPublisher(taskRepo, natsPublisher),
		OTPSentConsumer:     worker.NewOTPSentConsumer(natsSubscriber, taskRepo),
		OTPSentProcessor:    worker.NewOTPSentProcessor(taskRepo, otpService, bot),
	}, nil
}

func (a *App) Run(ctx context.Context, cfg *config.Config) error {
	workers := map[string]func(context.Context) error{
		"OTPRequestPublisher": a.OTPRequestPublisher.Run,
		"OTPSentConsumer":     a.OTPSentConsumer.Run,
		"OTPSentProcessor":    a.OTPSentProcessor.Run,
	}
	for name, run := range workers {
		go func() {
			if err := run(ctx); err != nil {
				slog.Error("воркер завершился с ошибкой", "worker", name, "err", err)
			}
		}()
	}

	if err := a.TGHandler.Run(ctx); err != nil {
		return fmt.Errorf("tg-bot: %w", err)
	}
	return nil
}
