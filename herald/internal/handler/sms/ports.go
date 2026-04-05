package smshandler

import (
	"context"
	"herald/internal/domain"

	"github.com/google/uuid"
)

type SMSTaskRepository interface {
	ListPending(ctx context.Context, limit int) ([]domain.SMSTask, error)
	Ack(ctx context.Context, id uuid.UUID, success bool, errMsg string) error
}
