package session

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

type Service struct {
	sessions sessionRepository
}

func NewService(sessions sessionRepository) *Service {
	return &Service{sessions: sessions}
}

func (s *Service) Create(ctx context.Context, p domain.CreateSessionParams) (uuid.UUID, error) {
	id, err := s.sessions.Create(ctx, p)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create session: %w", err)
	}
	return id, nil
}
