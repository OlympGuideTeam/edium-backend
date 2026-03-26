package worker

import (
	"context"
	"doorman/internal/domain"
	natsinf "doorman/internal/infra/nats"
	"encoding/json"
	"fmt"
	"log/slog"
)

type taskScheduler interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
}

type OTPRequestConsumer struct {
	subscriber *natsinf.Subscriber
	tasks      taskScheduler
}

func NewOTPRequestConsumer(subscriber *natsinf.Subscriber, tasks taskScheduler) *OTPRequestConsumer {
	return &OTPRequestConsumer{subscriber: subscriber, tasks: tasks}
}

func (c *OTPRequestConsumer) Run(ctx context.Context) error {
	slog.Info("otp-request-consumer: подписка", "subject", natsinf.SubjectOTPRequest)
	return c.subscriber.QueueSubscribe(ctx, natsinf.SubjectOTPRequest, natsinf.QueueOTPRequest, c.handle)
}

type otpRequestMsg struct {
	Phone   string         `json:"phone"`
	Channel domain.Channel `json:"channel"`
}

func (c *OTPRequestConsumer) handle(ctx context.Context, data []byte) error {
	var msg otpRequestMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}
	slog.InfoContext(ctx, "otp-request-consumer: получено", "phone", msg.Phone, "channel", msg.Channel)
	return c.tasks.Schedule(ctx, domain.OTPRequest, data)
}
