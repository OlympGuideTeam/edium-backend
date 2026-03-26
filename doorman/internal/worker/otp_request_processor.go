package worker

import (
	"context"
	"doorman/internal/domain"
	"doorman/internal/pkg/correlation"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

const (
	otpRequestPollInterval = 2 * time.Second
	otpRequestBatchSize    = 10
	otpRequestRetryAfter   = 30 * time.Second
)

type otpSender interface {
	SendOTP(ctx context.Context, phone string, channel domain.Channel) error
}

type OTPRequestProcessor struct {
	tasks   taskRepository
	service otpSender
}

func NewOTPRequestProcessor(tasks taskRepository, service otpSender) *OTPRequestProcessor {
	return &OTPRequestProcessor{tasks: tasks, service: service}
}

func (w *OTPRequestProcessor) Run(ctx context.Context) error {
	log.Printf("[otp-request-processor] started, interval=%s", otpRequestPollInterval)
	ticker := time.NewTicker(otpRequestPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				log.Printf("[otp-request-processor] batch error: %v", err)
			}
		}
	}
}

func (w *OTPRequestProcessor) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.OTPRequest, otpRequestBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for _, t := range tasks {
		if err := w.processTask(ctx, t); err != nil {
			log.Printf("[otp-request-processor] task_id=%s error: %v", t.ID, err)
			_ = w.tasks.MarkFailed(ctx, t.ID, err.Error(), otpRequestRetryAfter)
		}
	}
	return nil
}

func (w *OTPRequestProcessor) processTask(ctx context.Context, t domain.Task) error {
	var payload struct {
		Phone         string         `json:"phone"`
		Channel       domain.Channel `json:"channel"`
		CorrelationID string         `json:"correlation_id,omitempty"`
	}
	if err := json.Unmarshal(t.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	if payload.CorrelationID != "" {
		ctx = correlation.WithID(ctx, payload.CorrelationID)
	}

	log.Printf("[otp-request-processor] task_id=%s correlation_id=%s phone=%s",
		t.ID, payload.CorrelationID, payload.Phone)

	if err := w.service.SendOTP(ctx, payload.Phone, payload.Channel); err != nil {
		return err
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
