package user

import (
	"caesar/internal/domain"
	"context"

	"github.com/google/uuid"
)

type userService interface {
	GetMe(ctx context.Context, userID uuid.UUID) (*domain.User, error)
	UpdateMe(ctx context.Context, userID uuid.UUID, name, surname *string) error
	DeleteMe(ctx context.Context, userID uuid.UUID) error
	GetMeStatistic(ctx context.Context, userID uuid.UUID) (*domain.UserStatistic, error)
}
