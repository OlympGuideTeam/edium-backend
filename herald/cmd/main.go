package main

import (
	"context"
	"herald/internal/app"
	"herald/internal/config"
	"log"
	"os/signal"
	"syscall"
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

	go func() {
		if err := application.OTPRequestPublisher.Run(ctx); err != nil {
			log.Printf("OTPRequestPublisher завершился: %v", err)
		}
	}()

	go func() {
		if err := application.OTPSentConsumer.Run(ctx); err != nil {
			log.Printf("OTPSentConsumer завершился: %v", err)
		}
	}()

	go func() {
		if err := application.OTPDeliveryWorker.Run(ctx); err != nil {
			log.Printf("OTPDeliveryWorker завершился: %v", err)
		}
	}()

	// Telegram long polling — основной цикл.
	if err := application.TGHandler.Run(ctx); err != nil {
		log.Printf("TGHandler завершился: %v", err)
	}
}
