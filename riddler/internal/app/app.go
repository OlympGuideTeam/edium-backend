package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"riddler/internal/config"
	quizhandler "riddler/internal/handler/quiz"
	"riddler/internal/infra/db"
	"riddler/internal/infra/jwks"
	natsinf "riddler/internal/infra/nats"
	"riddler/internal/middleware"
	"riddler/internal/pkg/metrics"
	"riddler/internal/repository"
	quizsvc "riddler/internal/service/quiz"
)

type App struct {
	quizHandler *quizhandler.Handler

	jwksClient  *jwks.Client
	httpAddr    string
	serviceName string
}

func New(ctx context.Context, cfg *config.Config) (*App, error) {
	pgdb, err := db.NewDB(cfg.Postgres)
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

	quizRepo := repository.NewPgQuizRepository(pgdb)
	quizService := quizsvc.NewService(quizRepo)
	quizHandler := quizhandler.NewHandler(quizService)

	return &App{
		quizHandler: quizHandler,
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

	quizzes := api.Group("/quizzes")
	{
		quizzes.GET("", a.quizHandler.ListQuizzes)
		quizzes.POST("", a.quizHandler.CreateQuiz)
		quizzes.GET("/my", a.quizHandler.ListMyQuizzes)
		quizzes.GET("/:id", a.quizHandler.GetQuiz)
		quizzes.PATCH("/:id", a.quizHandler.UpdateQuiz)
		quizzes.POST("/:id/publish", a.quizHandler.PublishQuiz)
		quizzes.POST("/:id/copy", a.quizHandler.CopyQuiz)
		quizzes.PATCH("/:id/questions/order", a.quizHandler.ReorderQuestions)
		quizzes.POST("/:id/questions", a.quizHandler.AddQuestion)
		quizzes.DELETE("/:id/questions/:question_id", a.quizHandler.DeleteQuestion)
	}

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
