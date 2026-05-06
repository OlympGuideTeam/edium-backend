package worker

import (
	"context"
	"encoding/json"
	"fmt"
	natsinf "herald/internal/infra/nats"
	"log/slog"

	"github.com/google/uuid"
)

type userLogoutDeviceRepo interface {
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

type UserLogoutConsumer struct {
	subscriber *natsinf.Subscriber
	devices    userLogoutDeviceRepo
}

func NewUserLogoutConsumer(subscriber *natsinf.Subscriber, devices userLogoutDeviceRepo) *UserLogoutConsumer {
	return &UserLogoutConsumer{subscriber: subscriber, devices: devices}
}

func (c *UserLogoutConsumer) Run(ctx context.Context) error {
	slog.Info("user-logout-consumer: подписка", "subject", natsinf.SubjectUserLogout)
	return c.subscriber.QueueSubscribe(ctx, natsinf.SubjectUserLogout, natsinf.QueueUserLogout, c.handle)
}

type userLogoutMsg struct {
	UserID uuid.UUID `json:"user_id"`
}

func (c *UserLogoutConsumer) handle(ctx context.Context, data []byte) error {
	var msg userLogoutMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}

	slog.InfoContext(ctx, "user-logout-consumer: удаление fcm-устройств", "user_id", msg.UserID)

	if err := c.devices.DeleteByUserID(ctx, msg.UserID); err != nil {
		return fmt.Errorf("delete devices (user_id=%s): %w", msg.UserID, err)
	}
	return nil
}
