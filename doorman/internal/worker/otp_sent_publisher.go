package worker

import (
	"context"
	"doorman/internal/domain"
	natsinf "doorman/internal/infra/nats"
	"doorman/internal/infra/telemetry"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

const (
	otpSentPollInterval = 2 * time.Second
	otpSentBatchSize    = 10
	otpSentRetryAfter   = 30 * time.Second
)

type taskRepository interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
	ClaimPending(ctx context.Context, taskType domain.TaskType, limit int) ([]domain.Task, error)
	MarkDone(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string, retryAfter time.Duration) error
}

type OTPSentPublisher struct {
	tasks     taskRepository
	publisher *natsinf.Publisher
}

func NewOTPSentPublisher(tasks taskRepository, publisher *natsinf.Publisher) *OTPSentPublisher {
	return &OTPSentPublisher{tasks: tasks, publisher: publisher}
}

func (w *OTPSentPublisher) Run(ctx context.Context) error {
	slog.Info("otp-sent-publisher: запущен", "interval", otpSentPollInterval)
	ticker := time.NewTicker(otpSentPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("otp-sent-publisher: ошибка батча", "err", err)
			}
		}
	}
}

type otpSentPayload struct {
	Phone     string         `json:"phone"`
	OTP       uint64         `json:"otp"`
	Channel   domain.Channel `json:"channel"`
	ErrorCode string         `json:"error_code,omitempty"`
}

func (w *OTPSentPublisher) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.OTPSent, otpSentBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("otp-sent-publisher: ошибка задачи", "task_id", t.ID, "err", err)
			if mfErr := w.tasks.MarkFailed(context.WithoutCancel(ctx), t.ID, err.Error(), otpSentRetryAfter); mfErr != nil {
				slog.Error("otp-sent-publisher: не удалось сохранить ошибку задачи", "task_id", t.ID, "err", mfErr)
			}
		}
	}
	return nil
}

func (w *OTPSentPublisher) processTask(ctx context.Context, t domain.Task) error {
	var payload otpSentPayload
	if err := json.Unmarshal(t.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("doorman").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.otp_sent_publisher")
	defer span.End()

	slog.InfoContext(ctx, "otp-sent-publisher: публикация", "task_id", t.ID)

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	if err := w.publisher.Publish(ctx, natsinf.SubjectOTPSent, data); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
