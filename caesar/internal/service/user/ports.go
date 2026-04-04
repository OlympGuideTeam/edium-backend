package user

import (
	"caesar/internal/domain"
	"context"

	"github.com/google/uuid"
)

type userStore interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}
