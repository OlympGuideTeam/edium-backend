package session

import (
	"context"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

type sessionRepository interface {
	Create(ctx context.Context, p domain.CreateSessionParams) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.QuizSession, error)
	HasAttempts(ctx context.Context, sessionID uuid.UUID) (bool, error)
	HasSessions(ctx context.Context, quizTemplateID uuid.UUID) (bool, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
