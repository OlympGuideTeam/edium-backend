package user

import (
	"caesar/internal/domain"
	"context"

	"github.com/google/uuid"
)

type userService interface {
	GetMe(ctx context.Context, userID uuid.UUID) (*domain.User, error)
}
