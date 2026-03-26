package main

import (
	"context"
	"doorman/internal/app"
	"doorman/internal/config"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Запускаем воркеры в фоне.
	go func() {
		if err := application.OTPRequestConsumer.Run(ctx); err != nil {
			log.Printf("OTPRequestConsumer завершился: %v", err)
		}
	}()
	go func() {
		if err := application.OTPSentPublisher.Run(ctx); err != nil {
			log.Printf("OTPSentPublisher завершился: %v", err)
		}
	}()
	go func() {
		if err := application.UserCreatedPublisher.Run(ctx); err != nil {
			log.Printf("UserCreatedPublisher завершился: %v", err)
		}
	}()
	go func() {
		if err := application.OTPRequestProcessor.Run(ctx); err != nil {
			log.Printf("OTPRequestProcessor завершился: %v", err)
		}
	}()
	go func() {
		if err := application.UserDeletedConsumer.Run(ctx); err != nil {
			log.Printf("UserDeletedConsumer завершился: %v", err)
		}
	}()
	go func() {
		if err := application.UserDeletedProcessor.Run(ctx); err != nil {
			log.Printf("UserDeletedProcessor завершился: %v", err)
		}
	}()

	r := gin.Default()

	api := r.Group("/api/v1")

	api.POST("/otp/send", application.OtpHandler.Send)
	api.POST("/otp/verify", application.OtpHandler.Verify)

	api.POST("/auth/register", application.RegistrationHandler.Register)

	api.POST("/auth/refresh", application.TokenHandler.Refresh)
	api.POST("/auth/logout", application.TokenHandler.Logout)

	api.GET("/.well-known/jwks.json", application.KeyHandler.GetJWKS)

	if err = r.Run(fmt.Sprintf(":%d", cfg.App.Port)); err != nil {
		log.Fatal(err)
	}
}
