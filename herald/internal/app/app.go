package app

import (
	"herald/internal/config"
	"herald/internal/infra/db"
	natsinf "herald/internal/infra/nats"
	tginfra "herald/internal/infra/telegram"
	"herald/internal/repository"
	otpsvc "herald/internal/service/otp"
	tgbot "herald/internal/bot/telegram"
	"herald/internal/worker"
)

type App struct {
	TGHandler          *tgbot.Handler
	OTPRequestPublisher *worker.OTPRequestPublisher
	OTPSentConsumer    *worker.OTPSentConsumer
	OTPDeliveryWorker  *worker.OTPDeliveryWorker
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
		OTPSentConsumer:     worker.NewOTPSentConsumer(natsSubscriber, otpService),
		OTPDeliveryWorker:   worker.NewOTPDeliveryWorker(taskRepo, bot),
	}, nil
}
