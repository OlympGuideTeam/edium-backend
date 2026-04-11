package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	tgbot "herald/internal/bot/telegram"
	"herald/internal/config"
	"herald/internal/domain"
	smshandler "herald/internal/handler/sms"
	"herald/internal/infra/db"
	natsinf "herald/internal/infra/nats"
	smsinf "herald/internal/infra/sms"
	tginfra "herald/internal/infra/telegram"
	"herald/internal/repository"
	otpsvc "herald/internal/service/otp"
	"herald/internal/worker"

	"herald/internal/pkg/metrics"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type App struct {
	TGHandler           *tgbot.Handler
	OTPRequestPublisher *worker.OTPRequestPublisher
	OTPSentConsumer     *worker.OTPSentConsumer
	OTPSentProcessor    *worker.OTPSentProcessor
	smsHandler          *smshandler.Handler // nil если SMS не настроен
	httpAddr            string
	serviceName         string
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

	tgBot, err := tginfra.New(cfg.Telegram)
	if err != nil {
		return nil, err
	}

	txManager := db.NewTxManager(pgdb)
	taskRepo := repository.NewPgTaskRepository(pgdb)
	pendingOTPRepo := repository.NewPgPendingOTPRepository(pgdb)
	smsTaskRepo := repository.NewPgSMSTaskRepository(pgdb)

	otpService := otpsvc.NewService(txManager, taskRepo, pendingOTPRepo)

	natsPublisher := natsinf.NewPublisher(natsConn)
	natsSubscriber := natsinf.NewSubscriber(natsConn)

	senders := map[domain.Channel]worker.MessageSender{
		domain.ChannelTG: tginfra.NewSender(tgBot),
	}

	// SMS-отправитель: активен только если задан API-ключ.
	var smsSender worker.SMSSender
	var sh *smshandler.Handler
	if cfg.SMS.APIKey != "" {
		smsSender = smsinf.NewSender(smsTaskRepo, cfg.SMS.AllowedPhones)
		sh = smshandler.NewHandler(smsTaskRepo, cfg.SMS.APIKey)
		slog.Info("sms-шлюз: активирован", "allowed_phones", cfg.SMS.AllowedPhones)
	}

	return &App{
		TGHandler:           tgbot.NewHandler(tgBot, otpService),
		OTPRequestPublisher: worker.NewOTPRequestPublisher(taskRepo, natsPublisher),
		OTPSentConsumer:     worker.NewOTPSentConsumer(natsSubscriber, taskRepo),
		OTPSentProcessor:    worker.NewOTPSentProcessor(taskRepo, otpService, senders, smsSender),
		smsHandler:          sh,
		httpAddr:            fmt.Sprintf(":%d", cfg.App.Port),
		serviceName:         cfg.OTel.ServiceName,
	}, nil
}

func (a *App) Router() *gin.Engine {
	r := gin.Default()
	r.Use(otelgin.Middleware(a.serviceName))
	r.Use(metrics.Middleware(a.serviceName))
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	api := r.Group("/herald/v1")
	if a.smsHandler != nil {
		a.smsHandler.Register(api)
	}

	return r
}

func (a *App) Run(ctx context.Context) error {
	workers := map[string]func(context.Context) error{
		"OTPRequestPublisher": a.OTPRequestPublisher.Run,
		"OTPSentConsumer":     a.OTPSentConsumer.Run,
		"OTPSentProcessor":    a.OTPSentProcessor.Run,
		"TGHandler":           a.TGHandler.Run,
	}
	for name, run := range workers {
		go func() {
			if err := run(ctx); err != nil {
				slog.Error("воркер завершился с ошибкой", "worker", name, "err", err)
			}
		}()
	}

	srv := &http.Server{Addr: a.httpAddr, Handler: a.Router()}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.WithoutCancel(ctx))
	}()

	slog.Info("http-сервер: запущен", "addr", a.httpAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http-сервер: %w", err)
	}
	return nil
}
