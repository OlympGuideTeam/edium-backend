package user

import (
	"caesar/internal/domain"
	"caesar/internal/pkg/apperr"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type Service struct {
	users userStore
}

func NewService(users userStore) *Service {
	return &Service{users: users}
}

func (s *Service) GetMe(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("GetByID: %w", err)
	}
	if u == nil {
		return nil, apperr.New("USER_NOT_FOUND", "Пользователь не найден", 404)
	}
	if u.Status != domain.UserStatusActive {
		return nil, apperr.New("FORBIDDEN", "Пользователь заблокирован или удалён", 403)
	}
	return u, nil
}
