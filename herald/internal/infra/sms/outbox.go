package sms

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// SMSTaskRepository — запись задачи в outbox-таблицу.
type SMSTaskRepository interface {
	Create(ctx context.Context, phone, text string, idempotencyKey uuid.UUID) error
}

// Sender реализует отправку SMS через Android-шлюз:
// записывает задачу в outbox, Android-приложение поллит и физически отправляет SMS.
type Sender struct {
	repo          SMSTaskRepository
	allowedPhones map[string]struct{}
	blockedPhones map[string]struct{}
}

func NewSender(repo SMSTaskRepository, allowedPhones, blockedPhones []string) *Sender {
	allowed := make(map[string]struct{}, len(allowedPhones))
	for _, p := range allowedPhones {
		allowed[p] = struct{}{}
	}
	blocked := make(map[string]struct{}, len(blockedPhones))
	for _, p := range blockedPhones {
		blocked[p] = struct{}{}
	}
	return &Sender{repo: repo, allowedPhones: allowed, blockedPhones: blocked}
}

func (s *Sender) SendSMS(ctx context.Context, phone, text string, idempotencyKey uuid.UUID) error {
	if len(s.allowedPhones) > 0 {
		if _, ok := s.allowedPhones[phone]; !ok {
			slog.WarnContext(ctx, "sms: телефон не в белом списке, пропускаем", "phone", phone)
			return nil
		}
	}
	if len(s.blockedPhones) > 0 {
		if _, ok := s.blockedPhones[phone]; ok {
			slog.WarnContext(ctx, "sms: телефон в чёрном списке, пропускаем", "phone", phone)
			return nil
		}
	}
	if err := s.repo.Create(ctx, phone, text, idempotencyKey); err != nil {
		return err
	}
	slog.InfoContext(ctx, "sms: задача создана", "phone", phone)
	return nil
}
