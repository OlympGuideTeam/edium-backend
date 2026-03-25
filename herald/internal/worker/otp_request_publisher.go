package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"herald/internal/domain"
	natsinf "herald/internal/infra/nats"
	"herald/internal/pkg/correlation"
	"log"
	"time"

	"github.com/google/uuid"
)

const (
	otpRequestPollInterval = 2 * time.Second
	otpRequestBatchSize    = 10
	otpRequestRetryAfter   = 30 * time.Second
)

type otpRequestTaskRepo interface {
	ClaimPending(ctx context.Context, taskType domain.TaskType, limit int) ([]domain.Task, error)
	MarkDone(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string, retryAfter time.Duration) error
}

type OTPRequestPublisher struct {
	tasks     otpRequestTaskRepo
	publisher *natsinf.Publisher
}

func NewOTPRequestPublisher(tasks otpRequestTaskRepo, publisher *natsinf.Publisher) *OTPRequestPublisher {
	return &OTPRequestPublisher{tasks: tasks, publisher: publisher}
}

func (w *OTPRequestPublisher) Run(ctx context.Context) error {
	log.Printf("[otp-request-publisher] started, interval=%s", otpRequestPollInterval)
	ticker := time.NewTicker(otpRequestPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				log.Printf("[otp-request-publisher] batch error: %v", err)
			}
		}
	}
}

type otpRequestPayload struct {
	Phone         string         `json:"phone"`
	Channel       domain.Channel `json:"channel"`
	CorrelationID string         `json:"correlation_id"`
}

func (w *OTPRequestPublisher) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.OTPRequest, otpRequestBatchSize)
	if err != nil {
		return fmt.Errorf("claim pending: %w", err)
	}
	for _, t := range tasks {
		if err := w.processTask(ctx, t); err != nil {
			log.Printf("[otp-request-publisher] task_id=%s error: %v", t.ID, err)
			_ = w.tasks.MarkFailed(ctx, t.ID, err.Error(), otpRequestRetryAfter)
		}
	}
	return nil
}

func (w *OTPRequestPublisher) processTask(ctx context.Context, t domain.Task) error {
	var payload otpRequestPayload
	if err := json.Unmarshal(t.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	pubCtx := ctx
	if payload.CorrelationID != "" {
		pubCtx = correlation.WithID(ctx, payload.CorrelationID)
	}

	log.Printf("[otp-request-publisher] task_id=%s correlation_id=%s", t.ID, payload.CorrelationID)

	if err := w.publisher.Publish(pubCtx, natsinf.SubjectOTPRequest, t.Payload); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
