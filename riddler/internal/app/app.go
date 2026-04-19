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
	attempthandler "riddler/internal/handler/attempt"
	quizhandler "riddler/internal/handler/quiz"
	"riddler/internal/infra/db"
	"riddler/internal/infra/jwks"
	natsinf "riddler/internal/infra/nats"
	"riddler/internal/middleware"
	"riddler/internal/pkg/metrics"
	"riddler/internal/repository"
	attemptsvc "riddler/internal/service/attempt"
	quizsvc "riddler/internal/service/quiz"
	sessionsvc "riddler/internal/service/session"
	"riddler/internal/worker"
)

type App struct {
	quizHandler    *quizhandler.Handler
	attemptHandler *attempthandler.Handler
	attemptService *attemptsvc.Service

	quizTemplateAttachedPublisher *worker.QuizTemplateAttachedPublisher
	courseSessionCreatedPublisher *worker.CourseSessionCreatedPublisher
	attemptCreatedPublisher       *worker.AttemptCreatedPublisher
	attemptScoredPublisher        *worker.AttemptScoredPublisher
	generationRequestedPublisher  *worker.GenerationRequestedPublisher
	generationCompletedConsumer   *worker.GenerationCompletedConsumer
	generationCompletedProcessor  *worker.GenerationCompletedProcessor
	courseSessionDeletedConsumer  *worker.CourseSessionDeletedConsumer
	courseSessionDeletedProcessor *worker.CourseSessionDeletedProcessor

	gradingRequestedPublisher *worker.GradingRequestedPublisher
	gradingCompletedConsumer  *worker.GradingCompletedConsumer
	gradingCompletedProcessor *worker.GradingCompletedProcessor

	jwksClient  *jwks.Client
	httpAddr    string
	serviceName string
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

	js, err := natsConn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("jetstream: %w", err)
	}
	if err := natsinf.EnsureGenerationStream(js); err != nil {
		return nil, fmt.Errorf("ensure generation stream: %w", err)
	}
	if err := natsinf.EnsureGenerationCompletedStream(js); err != nil {
		return nil, fmt.Errorf("ensure generation completed stream: %w", err)
	}

	jwksClient := jwks.NewClient(cfg.Doorman.JWKSEndpoint)
	if err := jwksClient.Load(ctx); err != nil {
		return nil, fmt.Errorf("загрузка JWKS: %w", err)
	}
	jwksClient.StartRefresh(ctx)

	txManager := db.NewTxManager(pgdb)

	quizRepo := repository.NewPgQuizRepository(pgdb)
	sessionRepo := repository.NewPgSessionRepository(pgdb)
	attemptRepo := repository.NewPgAttemptRepository(pgdb)
	taskRepo := repository.NewPgTaskRepository(pgdb)

	sessionService := sessionsvc.NewService(sessionRepo)
	quizService := quizsvc.NewService(quizRepo, sessionService, txManager, taskRepo)
	attemptService := attemptsvc.NewService(attemptRepo, sessionRepo, sessionRepo, quizRepo, quizRepo, txManager, taskRepo)

	quizHandler := quizhandler.NewHandler(quizService)
	attemptHandler := attempthandler.NewHandler(attemptService)

	natsPublisher := natsinf.NewPublisher(natsConn)
	natsJSPublisher := natsinf.NewJetStreamPublisher(js)
	natsSubscriber := natsinf.NewSubscriber(natsConn)
	natsJSSubscriber := natsinf.NewJetStreamSubscriber(js)

	return &App{
		quizHandler:    quizHandler,
		attemptHandler: attemptHandler,
		attemptService: attemptService,

		quizTemplateAttachedPublisher: worker.NewQuizTemplateAttachedPublisher(taskRepo, natsPublisher),
		courseSessionCreatedPublisher: worker.NewCourseSessionCreatedPublisher(taskRepo, natsPublisher),
		attemptCreatedPublisher:       worker.NewAttemptCreatedPublisher(taskRepo, natsPublisher),
		attemptScoredPublisher:        worker.NewAttemptScoredPublisher(taskRepo, natsPublisher),

		generationRequestedPublisher:  worker.NewGenerationRequestedPublisher(taskRepo, natsJSPublisher),
		generationCompletedConsumer:   worker.NewGenerationCompletedConsumer(natsJSSubscriber, taskRepo),
		generationCompletedProcessor:  worker.NewGenerationCompletedProcessor(taskRepo, quizService),
		courseSessionDeletedConsumer:  worker.NewCourseSessionDeletedConsumer(natsSubscriber, taskRepo),
		courseSessionDeletedProcessor: worker.NewCourseSessionDeletedProcessor(taskRepo, sessionRepo),

		gradingRequestedPublisher: worker.NewGradingRequestedPublisher(taskRepo, natsPublisher),
		gradingCompletedConsumer:  worker.NewGradingCompletedConsumer(natsSubscriber, taskRepo),
		gradingCompletedProcessor: worker.NewGradingCompletedProcessor(taskRepo, attemptService),

		jwksClient:  jwksClient,
		httpAddr:    fmt.Sprintf(":%d", cfg.App.Port),
		serviceName: cfg.OTel.ServiceName,
	}, nil
}

func (a *App) workers() map[string]func(context.Context) error {
	return map[string]func(context.Context) error{
		"quiz-template-attached-publisher": a.quizTemplateAttachedPublisher.Run,
		"course-session-created-publisher": a.courseSessionCreatedPublisher.Run,
		"attempt-created-publisher":        a.attemptCreatedPublisher.Run,
		"attempt-scored-publisher":         a.attemptScoredPublisher.Run,
		"generation-requested-publisher":   a.generationRequestedPublisher.Run,
		"generation-completed-consumer":    a.generationCompletedConsumer.Run,
		"generation-completed-processor":   a.generationCompletedProcessor.Run,
		"course-session-deleted-consumer":  a.courseSessionDeletedConsumer.Run,
		"course-session-deleted-processor": a.courseSessionDeletedProcessor.Run,
		"grading-requested-publisher":      a.gradingRequestedPublisher.Run,
		"grading-completed-consumer":       a.gradingCompletedConsumer.Run,
		"grading-completed-processor":      a.gradingCompletedProcessor.Run,
	}
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
		quizzes.POST("/:id/generate", a.quizHandler.GenerateQuestions)
	}

	sessions := api.Group("/sessions")
	{
		sessions.POST("/test", a.quizHandler.CreateTestCourseSession)
		sessions.POST("/live", a.quizHandler.CreateLiveCourseSession)
		sessions.POST("/:session_id/attempts", a.attemptHandler.CreateAttempt)
		sessions.GET("/:session_id/attempts", a.attemptHandler.ListSessionAttempts)
	}

	attempts := api.Group("/attempts")
	{
		attempts.POST("/:attempt_id/answers", a.attemptHandler.SubmitAnswer)
		attempts.POST("/:attempt_id/finish", a.attemptHandler.Finish)
		attempts.GET("/:attempt_id/review", a.attemptHandler.GetAttemptReview)
		attempts.POST("/:attempt_id/submissions/:submission_id/grade", a.attemptHandler.GradeSubmission)
		attempts.POST("/:attempt_id/complete", a.attemptHandler.CompleteAttempt)
	}

	return r
}

func (a *App) Run(ctx context.Context) error {
	for name, run := range a.workers() {
		go func() {
			if err := run(ctx); err != nil {
				slog.Error("воркер завершился с ошибкой", "worker", name, "err", err)
			}
		}()
	}

	go a.attemptService.RunExpiryWorker(ctx)
	go a.attemptService.RunSessionGradingWorker(ctx)

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
