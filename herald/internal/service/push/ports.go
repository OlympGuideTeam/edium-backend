package pushsvc

import (
	"context"
	"herald/internal/domain"

	"github.com/google/uuid"
)

type FCMDeviceRepository interface {
	Upsert(ctx context.Context, userID uuid.UUID, fcmToken, platform string) error
	Delete(ctx context.Context, fcmToken string) error
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.FCMDevice, error)
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	DeleteTokens(ctx context.Context, tokens []string) error
}

type NotificationRepository interface {
	Save(ctx context.Context, n *domain.Notification) error
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Notification, error)
	MarkRead(ctx context.Context, id, userID uuid.UUID) error
	CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
}

// PushSender отправляет FCM push-уведомления.
// Возвращает список токенов, которые больше не действительны и должны быть удалены.
type PushSender interface {
	Send(ctx context.Context, tokens []string, title, body string, data map[string]string) (invalidTokens []string, err error)
	SendBadge(ctx context.Context, tokens []string, badge int) (invalidTokens []string, err error)
}
