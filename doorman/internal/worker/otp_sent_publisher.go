package worker

import (
	"context"
	"doorman/internal/domain"
	natsinf "doorman/internal/infra/nats"
	"doorman/internal/pkg/correlation"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

const (
	otpSentPollInterval = 2 * time.Second
	otpSentBatchSize    = 10
	otpSentRetryAfter   = 30 * time.Second
)

type taskRepository interface {
	ClaimPending(ctx context.Context, taskType domain.TaskType, limit int) ([]domain.Task, error)
	MarkDone(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string, retryAfter time.Duration) error
}

// OTPSentPublisher читает задачи otp_sent из БД и публикует их в Herald через NATS.
type OTPSentPublisher struct {
	tasks     taskRepository
	publisher *natsinf.Publisher
}

func NewOTPSentPublisher(tasks taskRepository, publisher *natsinf.Publisher) *OTPSentPublisher {
	return &OTPSentPublisher{tasks: tasks, publisher: publisher}
}

// Run запускает polling-цикл до отмены ctx.
func (w *OTPSentPublisher) Run(ctx context.Context) error {
	log.Printf("[otp-sent-publisher] запуск, интервал=%s", otpSentPollInterval)
	ticker := time.NewTicker(otpSentPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				log.Printf("[otp-sent-publisher] ошибка обработки батча: %v", err)
			}
		}
	}
}

type otpSentPayload struct {
	Phone         string         `json:"phone"`
	OTP           uint64         `json:"otp"`
	Channel       domain.Channel `json:"channel"`
	CorrelationID string         `json:"correlation_id,omitempty"`
}

func (w *OTPSentPublisher) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.OTPSent, otpSentBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}

	for _, t := range tasks {
		if err := w.processTask(ctx, t); err != nil {
			log.Printf("[otp-sent-publisher] task_id=%s ошибка: %v", t.ID, err)
			_ = w.tasks.MarkFailed(ctx, t.ID, err.Error(), otpSentRetryAfter)
		}
	}
	return nil
}

func (w *OTPSentPublisher) processTask(ctx context.Context, t domain.Task) error {
	var payload otpSentPayload
	if err := json.Unmarshal(t.Payload, &payload); err != nil {
		return fmt.Errorf("разбор payload: %w", err)
	}

	pubCtx := ctx
	if payload.CorrelationID != "" {
		pubCtx = correlation.WithID(ctx, payload.CorrelationID)
	}

	log.Printf("[otp-sent-publisher] task_id=%s correlation_id=%s phone=%s",
		t.ID, payload.CorrelationID, payload.Phone)

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("сериализация: %w", err)
	}

	if err := w.publisher.Publish(pubCtx, natsinf.SubjectOTPSent, data); err != nil {
		return fmt.Errorf("публикация: %w", err)
	}

	return w.tasks.MarkDone(ctx, t.ID)
}
