package worker

import (
	"context"
	"encoding/json"
	"fmt"
	natsinf "herald/internal/infra/nats"
	"herald/internal/pkg/correlation"
	"log"
)

type otpSentService interface {
	HandleOTPSent(ctx context.Context, correlationID string, otp uint64) error
}

type OTPSentConsumer struct {
	subscriber *natsinf.Subscriber
	service    otpSentService
}

func NewOTPSentConsumer(subscriber *natsinf.Subscriber, service otpSentService) *OTPSentConsumer {
	return &OTPSentConsumer{subscriber: subscriber, service: service}
}

func (c *OTPSentConsumer) Run(ctx context.Context) error {
	log.Printf("[otp-sent-consumer] subscribing to %s", natsinf.SubjectOTPSent)
	return c.subscriber.QueueSubscribe(ctx, natsinf.SubjectOTPSent, natsinf.QueueOTPSent, c.handle)
}

type otpSentMsg struct {
	Phone         string `json:"phone"`
	OTP           uint64 `json:"otp"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

func (c *OTPSentConsumer) handle(ctx context.Context, data []byte) error {
	var msg otpSentMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}

	if msg.CorrelationID != "" {
		ctx = correlation.WithID(ctx, msg.CorrelationID)
	}

	log.Printf("[otp-sent-consumer] correlation_id=%s", correlation.IDFromContext(ctx))
	return c.service.HandleOTPSent(ctx, msg.CorrelationID, msg.OTP)
}
