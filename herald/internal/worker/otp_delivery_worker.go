package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"herald/internal/domain"
	"herald/internal/pkg/correlation"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

const (
	otpDeliveryPollInterval = 2 * time.Second
	otpDeliveryBatchSize    = 10
	otpDeliveryRetryAfter   = 15 * time.Second
)

type otpDeliveryTaskRepo interface {
	ClaimPending(ctx context.Context, taskType domain.TaskType, limit int) ([]domain.Task, error)
	MarkDone(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string, retryAfter time.Duration) error
}

type OTPDeliveryWorker struct {
	tasks otpDeliveryTaskRepo
	bot   *tgbotapi.BotAPI
}

func NewOTPDeliveryWorker(tasks otpDeliveryTaskRepo, bot *tgbotapi.BotAPI) *OTPDeliveryWorker {
	return &OTPDeliveryWorker{tasks: tasks, bot: bot}
}

func (w *OTPDeliveryWorker) Run(ctx context.Context) error {
	log.Printf("[otp-delivery-worker] started, interval=%s", otpDeliveryPollInterval)
	ticker := time.NewTicker(otpDeliveryPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				log.Printf("[otp-delivery-worker] batch error: %v", err)
			}
		}
	}
}

type otpDeliveryPayload struct {
	ChatID        int64  `json:"chat_id"`
	OTP           uint64 `json:"otp"`
	CorrelationID string `json:"correlation_id"`
}

func (w *OTPDeliveryWorker) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.OTPDelivery, otpDeliveryBatchSize)
	if err != nil {
		return fmt.Errorf("claim pending: %w", err)
	}
	for _, t := range tasks {
		if err := w.processTask(ctx, t); err != nil {
			log.Printf("[otp-delivery-worker] task_id=%s error: %v", t.ID, err)
			_ = w.tasks.MarkFailed(ctx, t.ID, err.Error(), otpDeliveryRetryAfter)
		}
	}
	return nil
}

func (w *OTPDeliveryWorker) processTask(ctx context.Context, t domain.Task) error {
	var payload otpDeliveryPayload
	if err := json.Unmarshal(t.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	if payload.CorrelationID != "" {
		ctx = correlation.WithID(ctx, payload.CorrelationID)
	}

	log.Printf("[otp-delivery-worker] task_id=%s correlation_id=%s chat_id=%d",
		t.ID, correlation.IDFromContext(ctx), payload.ChatID)

	msg := tgbotapi.NewMessage(payload.ChatID, fmt.Sprintf("Ваш код: %06d", payload.OTP))
	if _, err := w.bot.Send(msg); err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
