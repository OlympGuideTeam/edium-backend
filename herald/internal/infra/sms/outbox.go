package sms

import (
	"context"
	"fmt"
)

// SMSTaskRepository — запись задачи в outbox-таблицу.
type SMSTaskRepository interface {
	Create(ctx context.Context, phone, text string) error
}

// Sender реализует отправку SMS через Android-шлюз:
// записывает задачу в outbox, Android-приложение поллит и физически отправляет SMS.
type Sender struct {
	repo          SMSTaskRepository
	allowedPhones map[string]struct{}
}

func NewSender(repo SMSTaskRepository, allowedPhones []string) *Sender {
	allowed := make(map[string]struct{}, len(allowedPhones))
	for _, p := range allowedPhones {
		allowed[p] = struct{}{}
	}
	return &Sender{repo: repo, allowedPhones: allowed}
}

func (s *Sender) SendSMS(ctx context.Context, phone, text string) error {
	if len(s.allowedPhones) > 0 {
		if _, ok := s.allowedPhones[phone]; !ok {
			return fmt.Errorf("телефон не в белом списке SMS: %s", phone)
		}
	}
	return s.repo.Create(ctx, phone, text)
}
