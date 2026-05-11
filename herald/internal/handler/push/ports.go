package pushhandler

import (
	"context"
	"herald/internal/domain"

	"github.com/google/uuid"
)

type PushService interface {
	RegisterDevice(ctx context.Context, userID uuid.UUID, fcmToken, platform string) error
	DeleteDevice(ctx context.Context, fcmToken string) error
	ListNotifications(ctx context.Context, userID uuid.UUID) ([]domain.Notification, error)
	MarkRead(ctx context.Context, userID, notifID uuid.UUID) error
	CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
}
