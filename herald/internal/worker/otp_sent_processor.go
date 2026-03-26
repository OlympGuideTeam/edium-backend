package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"herald/internal/domain"
	"herald/internal/infra/telemetry"
	"log/slog"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

const (
	otpSentPollInterval = 2 * time.Second
	otpSentBatchSize    = 10
	otpSentRetryAfter   = 30 * time.Second
)

type otpSentTaskRepo interface {
	ClaimPending(ctx context.Context, taskType domain.TaskType, limit int) ([]domain.Task, error)
	MarkDone(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string, retryAfter time.Duration) error
}

type pendingOTPLookup interface {
	GetPendingOTP(ctx context.Context, phone string) (*domain.PendingOTP, error)
	DeletePendingOTP(ctx context.Context, phone string) error
}

type OTPSentProcessor struct {
	tasks      otpSentTaskRepo
	pendingOTP pendingOTPLookup
	bot        *tgbotapi.BotAPI
}

func NewOTPSentProcessor(tasks otpSentTaskRepo, pendingOTP pendingOTPLookup, bot *tgbotapi.BotAPI) *OTPSentProcessor {
	return &OTPSentProcessor{tasks: tasks, pendingOTP: pendingOTP, bot: bot}
}

func (w *OTPSentProcessor) Run(ctx context.Context) error {
	slog.Info("otp-sent-processor: запущен", "interval", otpSentPollInterval)
	ticker := time.NewTicker(otpSentPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("otp-sent-processor: ошибка батча", "err", err)
			}
		}
	}
}

func (w *OTPSentProcessor) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.OTPSent, otpSentBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("otp-sent-processor: ошибка задачи", "task_id", t.ID, "err", err)
			_ = w.tasks.MarkFailed(ctx, t.ID, err.Error(), otpSentRetryAfter)
		}
	}
	return nil
}

func (w *OTPSentProcessor) processTask(ctx context.Context, t domain.Task) error {
	var payload struct {
		Phone string `json:"phone"`
		OTP   uint64 `json:"otp"`
	}
	if err := json.Unmarshal(t.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("herald").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.otp_sent_processor")
	defer span.End()

	slog.InfoContext(ctx, "otp-sent-processor: обработка", "task_id", t.ID, "phone", payload.Phone)

	pending, err := w.pendingOTP.GetPendingOTP(ctx, payload.Phone)
	if err != nil {
		return fmt.Errorf("GetPendingOTP: %w", err)
	}
	if pending == nil {
		slog.InfoContext(ctx, "otp-sent-processor: pending_otp не найден или истёк", "phone", payload.Phone)
		return w.tasks.MarkDone(ctx, t.ID)
	}

	msg := tgbotapi.NewMessage(pending.ChatID, fmt.Sprintf("Ваш код: %06d", payload.OTP))
	if _, err := w.bot.Send(msg); err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}

	if err := w.pendingOTP.DeletePendingOTP(ctx, payload.Phone); err != nil {
		return fmt.Errorf("DeletePendingOTP: %w", err)
	}

	return w.tasks.MarkDone(ctx, t.ID)
}
