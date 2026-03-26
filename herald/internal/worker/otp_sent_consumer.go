package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"herald/internal/domain"
	natsinf "herald/internal/infra/nats"
	"log/slog"
)

type otpSentScheduler interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
}

type OTPSentConsumer struct {
	subscriber *natsinf.Subscriber
	tasks      otpSentScheduler
}

func NewOTPSentConsumer(subscriber *natsinf.Subscriber, tasks otpSentScheduler) *OTPSentConsumer {
	return &OTPSentConsumer{subscriber: subscriber, tasks: tasks}
}

func (c *OTPSentConsumer) Run(ctx context.Context) error {
	slog.Info("otp-sent-consumer: подписка", "subject", natsinf.SubjectOTPSent)
	return c.subscriber.QueueSubscribe(ctx, natsinf.SubjectOTPSent, natsinf.QueueOTPSent, c.handle)
}

type otpSentMsg struct {
	Phone string `json:"phone"`
	OTP   uint64 `json:"otp"`
}

func (c *OTPSentConsumer) handle(ctx context.Context, data []byte) error {
	var msg otpSentMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}
	slog.InfoContext(ctx, "otp-sent-consumer: получено", "phone", msg.Phone)
	return c.tasks.Schedule(ctx, domain.OTPSent, data)
}
