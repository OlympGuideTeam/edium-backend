package session

import (
	"context"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

type sessionRepository interface {
	Create(ctx context.Context, p domain.CreateSessionParams) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.QuizSession, error)
	GetActiveTestSession(ctx context.Context, quizTemplateID uuid.UUID) (*domain.QuizSession, error)
}
