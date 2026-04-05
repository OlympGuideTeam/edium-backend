package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"herald/internal/domain"
	"herald/internal/infra/telemetry"
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

// MessageSender доставляет текстовое сообщение в конкретный чат бота.
type MessageSender interface {
	Send(ctx context.Context, chatID int64, text string) error
}

// SMSSender записывает SMS-задачу в outbox для Android-шлюза.
type SMSSender interface {
	SendSMS(ctx context.Context, phone string, text string) error
}

type otpSentTaskRepo interface {
	ClaimPending(ctx context.Context, taskType domain.TaskType, limit int) ([]domain.Task, error)
	MarkDone(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string, retryAfter time.Duration) error
}

type pendingOTPLookup interface {
	GetPendingOTP(ctx context.Context, phone string, channel domain.Channel) (*domain.PendingOTP, error)
	DeletePendingOTP(ctx context.Context, phone string, channel domain.Channel) error
}

type OTPSentProcessor struct {
	tasks      otpSentTaskRepo
	pendingOTP pendingOTPLookup
	senders    map[domain.Channel]MessageSender
	smsSender  SMSSender // nil если SMS не настроен
}

func NewOTPSentProcessor(tasks otpSentTaskRepo, pendingOTP pendingOTPLookup, senders map[domain.Channel]MessageSender, smsSender SMSSender) *OTPSentProcessor {
	return &OTPSentProcessor{tasks: tasks, pendingOTP: pendingOTP, senders: senders, smsSender: smsSender}
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
			if mfErr := w.tasks.MarkFailed(context.WithoutCancel(ctx), t.ID, err.Error(), otpSentRetryAfter); mfErr != nil {
				slog.Error("otp-sent-processor: не удалось сохранить ошибку задачи", "task_id", t.ID, "err", mfErr)
			}
		}
	}
	return nil
}

func (w *OTPSentProcessor) processTask(ctx context.Context, t domain.Task) error {
	var payload struct {
		Phone     string         `json:"phone"`
		OTP       uint64         `json:"otp"`
		Channel   domain.Channel `json:"channel"`
		ErrorCode string         `json:"error_code,omitempty"`
	}
	if err := json.Unmarshal(t.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("herald").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.otp_sent_processor")
	defer span.End()

	slog.InfoContext(ctx, "otp-sent-processor: обработка", "task_id", t.ID, "channel", payload.Channel)

	// SMS-канал: отправляем через Android-шлюз без pending_otp.
	if payload.Channel == domain.ChannelSMS {
		if w.smsSender == nil {
			return fmt.Errorf("SMS-отправитель не настроен (SMS_API_KEY не задан)")
		}
		var text string
		if payload.ErrorCode != "" {
			text = otpErrorMessage(payload.ErrorCode)
		} else {
			text = fmt.Sprintf("Ваш код Edium: %06d", payload.OTP)
		}
		if err := w.smsSender.SendSMS(ctx, payload.Phone, text); err != nil {
			return fmt.Errorf("send sms (phone=%s): %w", payload.Phone, err)
		}
		return w.tasks.MarkDone(ctx, t.ID)
	}

	pending, err := w.pendingOTP.GetPendingOTP(ctx, payload.Phone, payload.Channel)
	if err != nil {
		return fmt.Errorf("GetPendingOTP: %w", err)
	}
	if pending == nil {
		slog.InfoContext(ctx, "otp-sent-processor: pending_otp не найден или истёк", "phone", payload.Phone, "channel", payload.Channel)
		return w.tasks.MarkDone(ctx, t.ID)
	}

	sender, ok := w.senders[payload.Channel]
	if !ok {
		return fmt.Errorf("нет отправителя для канала %q", payload.Channel)
	}

	var text string
	if payload.ErrorCode != "" {
		text = otpErrorMessage(payload.ErrorCode)
	} else {
		text = fmt.Sprintf("Ваш код: %06d", payload.OTP)
	}

	if err := sender.Send(ctx, pending.ChatID, text); err != nil {
		return fmt.Errorf("send message (channel=%s): %w", payload.Channel, err)
	}

	if err := w.pendingOTP.DeletePendingOTP(ctx, payload.Phone, payload.Channel); err != nil {
		return fmt.Errorf("DeletePendingOTP: %w", err)
	}

	return w.tasks.MarkDone(ctx, t.ID)
}

func otpErrorMessage(code string) string {
	switch code {
	case "OTP_ALREADY_SENT":
		return "Код уже был отправлен и действует 3 минуты, затем возможна повторная отправка."
	case "PHONE_UNAVAILABLE":
		return "Ваш номер удалён или заблокирован. Обратитесь в поддержку."
	default:
		return "Произошла ошибка при отправке кода. Попробуйте позже."
	}
}
