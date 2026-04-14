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

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*domain.QuizSession, error) {
	session, err := s.sessions.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get session by id: %w", err)
	}
	return session, nil
}

func (s *Service) GetActiveTestSession(ctx context.Context, quizTemplateID uuid.UUID) (*domain.QuizSession, error) {
	session, err := s.sessions.GetActiveTestSession(ctx, quizTemplateID)
	if err != nil {
		return nil, fmt.Errorf("get active test session: %w", err)
	}
	return session, nil
}
