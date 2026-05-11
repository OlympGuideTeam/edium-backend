package pushsvc

import (
	"context"
	"herald/internal/domain"
	"log/slog"

	"github.com/google/uuid"
)

type Service struct {
	devices       FCMDeviceRepository
	notifications NotificationRepository
	sender        PushSender // nil если Firebase не настроен
}

func NewService(
	devices FCMDeviceRepository,
	notifications NotificationRepository,
	sender PushSender,
) *Service {
	return &Service{
		devices:       devices,
		notifications: notifications,
		sender:        sender,
	}
}

func (s *Service) RegisterDevice(ctx context.Context, userID uuid.UUID, fcmToken, platform string) error {
	return s.devices.Upsert(ctx, userID, fcmToken, platform)
}

func (s *Service) DeleteDevice(ctx context.Context, fcmToken string) error {
	return s.devices.Delete(ctx, fcmToken)
}

func (s *Service) ListNotifications(ctx context.Context, userID uuid.UUID) ([]domain.Notification, error) {
	return s.notifications.ListByUserID(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, userID, notifID uuid.UUID) error {
	if err := s.notifications.MarkRead(ctx, notifID, userID); err != nil {
		return err
	}

	if s.sender == nil {
		return nil
	}

	unread, err := s.notifications.CountUnread(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "push: не удалось посчитать unread", "user_id", userID, "err", err)
		return nil
	}

	devices, err := s.devices.ListByUserID(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "push: не удалось получить устройства", "user_id", userID, "err", err)
		return nil
	}
	if len(devices) == 0 {
		return nil
	}

	tokens := make([]string, len(devices))
	for i, d := range devices {
		tokens[i] = d.FCMToken
	}

	invalid, err := s.sender.SendBadge(ctx, tokens, unread)
	if err != nil {
		slog.ErrorContext(ctx, "push: не удалось отправить badge", "user_id", userID, "err", err)
		return nil
	}

	if len(invalid) > 0 {
		if err := s.devices.DeleteTokens(ctx, invalid); err != nil {
			slog.ErrorContext(ctx, "push: не удалось удалить невалидные токены", "err", err)
		}
	}

	return nil
}

func (s *Service) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.notifications.CountUnread(ctx, userID)
}

func (s *Service) DeleteUserDevices(ctx context.Context, userID uuid.UUID) error {
	return s.devices.DeleteByUserID(ctx, userID)
}
