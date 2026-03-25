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

type otpSender interface {
	SendOTP(ctx context.Context, phone string, channel domain.Channel) error
}

// OTPRequestConsumer читает запросы на отправку OTP из NATS и вызывает сервис.
type OTPRequestConsumer struct {
	subscriber *natsinf.Subscriber
	service    otpSender
}

func NewOTPRequestConsumer(subscriber *natsinf.Subscriber, service otpSender) *OTPRequestConsumer {
	return &OTPRequestConsumer{subscriber: subscriber, service: service}
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

	return c.service.SendOTP(ctx, msg.Phone, msg.Channel)
}
