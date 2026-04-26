package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"louvre/internal/config"
	imagehandler "louvre/internal/handler/image"
	"louvre/internal/infra/db"
	"louvre/internal/infra/jwks"
	"louvre/internal/infra/minio"
	"louvre/internal/middleware"
	"louvre/internal/pkg/metrics"
	"louvre/internal/repository"
	"louvre/internal/service"
)

type App struct {
	imageHandler *imagehandler.Handler
	jwksClient   *jwks.Client
	httpAddr     string
	serviceName  string
}

func New(cfg *config.Config) (*App, error) {
	jwksClient := jwks.NewClient(cfg.Doorman.JWKSEndpoint)
	if err := jwksClient.Load(context.Background()); err != nil {
		return nil, fmt.Errorf("загрузка JWKS: %w", err)
	}

	pgDB, err := db.NewDB(cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("подключение к PostgreSQL: %w", err)
	}

	minioClient, err := minio.NewClient(cfg.MinIO)
	if err != nil {
		return nil, fmt.Errorf("подключение к MinIO: %w", err)
	}

	imageRepo := repository.NewPgImageRepository(pgDB)

	allowedTypes := strings.Split(cfg.Image.AllowedMimeTypes, ",")
	imageService := service.NewImageService(
		imageRepo,
		minioClient,
		cfg.Image.MaxFileSize,
		cfg.Image.MaxWidth,
		cfg.Image.MaxHeight,
		cfg.Image.MaxUploadsPerHour,
		allowedTypes,
	)
	imageHandler := imagehandler.NewHandler(imageService)

	return &App{
		imageHandler: imageHandler,
		jwksClient:   jwksClient,
		httpAddr:     fmt.Sprintf(":%d", cfg.App.Port),
		serviceName:  cfg.OTel.ServiceName,
	}, nil
}

func (a *App) Router() *gin.Engine {
	r := gin.Default()
	r.Use(otelgin.Middleware(a.serviceName))
	r.Use(metrics.Middleware(a.serviceName))
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	auth := middleware.Auth(a.jwksClient)

	api := r.Group("/louvre/v1")
	api.Use(auth)

	images := api.Group("/images")
	{
		images.POST("/upload", a.imageHandler.Upload)
		images.GET("/:id", a.imageHandler.Download)
		images.DELETE("/:id", a.imageHandler.Delete)
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
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http-сервер: %w", err)
	}
	return nil
}
