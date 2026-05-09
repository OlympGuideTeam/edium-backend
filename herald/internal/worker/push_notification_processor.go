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
	pushPollInterval = 2 * time.Second
	pushBatchSize    = 10
	pushRetryAfter   = 30 * time.Second
)

type pushTaskRepo interface {
	ClaimPending(ctx context.Context, taskType domain.TaskType, limit int) ([]domain.Task, error)
	MarkDone(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string, retryAfter time.Duration) error
}

type pushFCMDeviceRepo interface {
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.FCMDevice, error)
	DeleteTokens(ctx context.Context, tokens []string) error
}

type pushNotificationRepo interface {
	Save(ctx context.Context, n *domain.Notification) error
	CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
}

// PushSender определён в service/push/ports.go и передаётся через интерфейс.
type pushSender interface {
	Send(ctx context.Context, tokens []string, title, body string, data map[string]string) (invalidTokens []string, err error)
	SendBadge(ctx context.Context, tokens []string, badge int) (invalidTokens []string, err error)
}

type PushNotificationProcessor struct {
	tasks         pushTaskRepo
	devices       pushFCMDeviceRepo
	notifications pushNotificationRepo
	sender        pushSender // nil если Firebase не настроен
}

func NewPushNotificationProcessor(
	tasks pushTaskRepo,
	devices pushFCMDeviceRepo,
	notifications pushNotificationRepo,
	sender pushSender,
) *PushNotificationProcessor {
	return &PushNotificationProcessor{
		tasks:         tasks,
		devices:       devices,
		notifications: notifications,
		sender:        sender,
	}
}

func (w *PushNotificationProcessor) Run(ctx context.Context) error {
	slog.Info("push-notification-processor: запущен", "interval", pushPollInterval)
	ticker := time.NewTicker(pushPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				slog.Error("push-notification-processor: ошибка батча", "err", err)
			}
		}
	}
}

func (w *PushNotificationProcessor) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.PushNotification, pushBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for i := range tasks {
		t := tasks[i]
		if err := w.processTask(ctx, t); err != nil {
			slog.Error("push-notification-processor: ошибка задачи", "task_id", t.ID, "err", err)
			if mfErr := w.tasks.MarkFailed(context.WithoutCancel(ctx), t.ID, err.Error(), pushRetryAfter); mfErr != nil {
				slog.Error("push-notification-processor: не удалось сохранить ошибку задачи", "task_id", t.ID, "err", mfErr)
			}
		}
	}
	return nil
}

type pushNotificationPayload struct {
	UserID uuid.UUID         `json:"user_id"`
	Title  string            `json:"title"`
	Body   string            `json:"body"`
	Data   map[string]string `json:"data,omitempty"`
}

func (w *PushNotificationProcessor) processTask(ctx context.Context, t domain.Task) error {
	var payload pushNotificationPayload
	if err := json.Unmarshal(t.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	ctx, span := otel.Tracer("herald").Start(telemetry.Extract(ctx, t.TraceCtx), "worker.push_notification_processor")
	defer span.End()

	slog.InfoContext(ctx, "push-notification-processor: обработка", "task_id", t.ID, "user_id", payload.UserID)

	notif := &domain.Notification{
		ID:     uuid.New(),
		UserID: payload.UserID,
		Title:  payload.Title,
		Body:   payload.Body,
	}
	if route, ok := payload.Data["route"]; ok {
		notif.Data = &domain.NotificationData{Route: &route}
		if role, ok := payload.Data["role"]; ok {
			notif.Data.Role = &role
		}
	}
	if err := w.notifications.Save(ctx, notif); err != nil {
		return fmt.Errorf("save notification: %w", err)
	}

	if w.sender == nil {
		slog.WarnContext(ctx, "push-notification-processor: Firebase не настроен, push не отправлен", "user_id", payload.UserID)
		return w.tasks.MarkDone(ctx, t.ID)
	}

	devices, err := w.devices.ListByUserID(ctx, payload.UserID)
	if err != nil {
		return fmt.Errorf("list devices: %w", err)
	}
	if len(devices) == 0 {
		return w.tasks.MarkDone(ctx, t.ID)
	}

	tokens := make([]string, len(devices))
	for i, d := range devices {
		tokens[i] = d.FCMToken
	}

	invalid, err := w.sender.Send(ctx, tokens, payload.Title, payload.Body, payload.Data)
	if err != nil {
		return fmt.Errorf("send push: %w", err)
	}

	if len(invalid) > 0 {
		if delErr := w.devices.DeleteTokens(ctx, invalid); delErr != nil {
			slog.ErrorContext(ctx, "push-notification-processor: не удалось удалить невалидные токены", "err", delErr)
		}
	}

	unread, err := w.notifications.CountUnread(ctx, payload.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "push-notification-processor: не удалось посчитать unread", "user_id", payload.UserID, "err", err)
	} else {
		badgeTokens := make([]string, len(devices))
		for i, d := range devices {
			badgeTokens[i] = d.FCMToken
		}
		if invalidBadge, badgeErr := w.sender.SendBadge(ctx, badgeTokens, unread); badgeErr != nil {
			slog.ErrorContext(ctx, "push-notification-processor: не удалось отправить badge", "user_id", payload.UserID, "err", badgeErr)
		} else if len(invalidBadge) > 0 {
			if delErr := w.devices.DeleteTokens(ctx, invalidBadge); delErr != nil {
				slog.ErrorContext(ctx, "push-notification-processor: не удалось удалить невалидные токены badge", "err", delErr)
			}
		}
	}

	return w.tasks.MarkDone(ctx, t.ID)
}
