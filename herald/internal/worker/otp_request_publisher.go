package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"herald/internal/domain"
	natsinf "herald/internal/infra/nats"
	"herald/internal/infra/telemetry"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
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
	slog.Info("otp-request-publisher: запущен", "interval", otpRequestPollInterval)
	ticker := time.NewTicker(otpRequestPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("otp-request-publisher: ошибка батча", "err", err)
			}
		}
	}
}

type otpRequestPayload struct {
	Phone   string         `json:"phone"`
	Channel domain.Channel `json:"channel"`
}

func (w *OTPRequestPublisher) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.OTPRequest, otpRequestBatchSize)
	if err != nil {
		return fmt.Errorf("claim pending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("otp-request-publisher: ошибка задачи", "task_id", t.ID, "err", err)
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

	ctx, span := otel.Tracer("herald").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.otp_request_publisher")
	defer span.End()

	slog.InfoContext(ctx, "otp-request-publisher: публикация", "task_id", t.ID, "phone", payload.Phone)

	if err := w.publisher.Publish(ctx, natsinf.SubjectOTPRequest, t.Payload); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
