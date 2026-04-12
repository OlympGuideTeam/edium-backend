package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"riddler/internal/config"
	quizhandler "riddler/internal/handler/quiz"
	"riddler/internal/infra/db"
	"riddler/internal/infra/jwks"
	natsinf "riddler/internal/infra/nats"
	"riddler/internal/middleware"
	"riddler/internal/pkg/metrics"
	quizsvc "riddler/internal/service/quiz"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type App struct {
	QuizHandler *quizhandler.Handler

	jwksClient  *jwks.Client
	httpAddr    string
	serviceName string
}

func New(ctx context.Context, cfg *config.Config) (*App, error) {
	_, err := db.NewDB(cfg.Postgres)
	if err != nil {
		return nil, err
	}

	_, err = natsinf.New(cfg.NATS)
	if err != nil {
		return nil, err
	}

	jwksClient := jwks.NewClient(cfg.Doorman.JWKSEndpoint)
	if err := jwksClient.Load(ctx); err != nil {
		return nil, fmt.Errorf("загрузка JWKS: %w", err)
	}
	jwksClient.StartRefresh(ctx)

	quizService := quizsvc.NewService()
	quizHandler := quizhandler.NewHandler(quizService)

	return &App{
		QuizHandler: quizHandler,
		jwksClient:  jwksClient,
		httpAddr:    fmt.Sprintf(":%d", cfg.App.Port),
		serviceName: cfg.OTel.ServiceName,
	}, nil
}

func (a *App) Router() *gin.Engine {
	r := gin.Default()
	r.Use(otelgin.Middleware(a.serviceName))
	r.Use(metrics.Middleware(a.serviceName))
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	auth := middleware.Auth(a.jwksClient)

	api := r.Group("/riddler/v1")
	api.Use(auth)

	// TODO: добавить маршруты

	return r
}

func (a *App) Run(ctx context.Context) error {
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
