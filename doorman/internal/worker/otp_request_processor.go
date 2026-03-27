package worker

import (
	"context"
	"doorman/internal/domain"
	"doorman/internal/infra/telemetry"
	otpsvc "doorman/internal/service/otp"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
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
	slog.Info("otp-request-processor: запущен", "interval", otpRequestPollInterval)
	ticker := time.NewTicker(otpRequestPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("otp-request-processor: ошибка батча", "err", err)
			}
		}
	}
}

func (w *OTPRequestProcessor) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.OTPRequest, otpRequestBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("otp-request-processor: ошибка задачи", "task_id", t.ID, "err", err)
			if mfErr := w.tasks.MarkFailed(context.WithoutCancel(ctx), t.ID, err.Error(), otpRequestRetryAfter); mfErr != nil {
				slog.Error("otp-request-processor: не удалось сохранить ошибку задачи", "task_id", t.ID, "err", mfErr)
			}
		}
	}
	return nil
}

func (w *OTPRequestProcessor) processTask(ctx context.Context, t domain.Task) error {
	var payload struct {
		Phone   string         `json:"phone"`
		Channel domain.Channel `json:"channel"`
	}
	if err := json.Unmarshal(t.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("doorman").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.otp_request_processor")
	defer span.End()

	slog.InfoContext(ctx, "otp-request-processor: обработка", "task_id", t.ID, "phone", payload.Phone)

	if err := w.service.SendOTP(ctx, payload.Phone, payload.Channel); err != nil {
		if errors.Is(err, otpsvc.ErrAlreadySent) || errors.Is(err, otpsvc.ErrPhoneUnavailable) {
			if schedErr := w.scheduleErrorNotification(ctx, payload.Phone, payload.Channel, err.Error()); schedErr != nil {
				return fmt.Errorf("scheduleErrorNotification: %w", schedErr)
			}
			return w.tasks.MarkDone(ctx, t.ID)
		}
		return err
	}
	return w.tasks.MarkDone(ctx, t.ID)
}

func (w *OTPRequestProcessor) scheduleErrorNotification(ctx context.Context, phone string, channel domain.Channel, errorCode string) error {
	p, err := json.Marshal(struct {
		Phone     string         `json:"phone"`
		OTP       uint64         `json:"otp"`
		Channel   domain.Channel `json:"channel"`
		ErrorCode string         `json:"error_code"`
	}{
		Phone:     phone,
		Channel:   channel,
		ErrorCode: errorCode,
	})
	if err != nil {
		return err
	}
	return w.tasks.Schedule(ctx, domain.OTPSent, p)
}
