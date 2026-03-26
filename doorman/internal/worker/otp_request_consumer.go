package worker

import (
	"context"
	"doorman/internal/domain"
	natsinf "doorman/internal/infra/nats"
	"doorman/internal/pkg/correlation"
	"encoding/json"
	"fmt"
	"log"
)

type taskScheduler interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
}

// OTPRequestConsumer читает запросы на отправку OTP из NATS и сохраняет задачу в БД.
type OTPRequestConsumer struct {
	subscriber *natsinf.Subscriber
	tasks      taskScheduler
}

func NewOTPRequestConsumer(subscriber *natsinf.Subscriber, tasks taskScheduler) *OTPRequestConsumer {
	return &OTPRequestConsumer{subscriber: subscriber, tasks: tasks}
}

// Run блокируется до отмены ctx.
func (c *OTPRequestConsumer) Run(ctx context.Context) error {
	log.Printf("[otp-request-consumer] подписка на %s", natsinf.SubjectOTPRequest)
	return c.subscriber.QueueSubscribe(ctx, natsinf.SubjectOTPRequest, natsinf.QueueOTPRequest, c.handle)
}

type otpRequestMsg struct {
	Phone         string         `json:"phone"`
	Channel       domain.Channel `json:"channel"`
	CorrelationID string         `json:"correlation_id,omitempty"`
}

func (c *OTPRequestConsumer) handle(ctx context.Context, data []byte) error {
	var msg otpRequestMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("разбор сообщения: %w", err)
	}

	if msg.CorrelationID != "" {
		ctx = correlation.WithID(ctx, msg.CorrelationID)
	}

	log.Printf("[otp-request-consumer] correlation_id=%s phone=%s channel=%s",
		correlation.IDFromContext(ctx), msg.Phone, msg.Channel)

	return c.tasks.Schedule(ctx, domain.OTPRequest, data)
}
