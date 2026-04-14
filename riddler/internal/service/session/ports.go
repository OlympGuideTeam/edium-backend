package session

import (
	"context"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

type sessionRepository interface {
	Create(ctx context.Context, p domain.CreateSessionParams) (uuid.UUID, error)
	GetActiveTestSession(ctx context.Context, quizTemplateID uuid.UUID) (*domain.QuizSession, error)
}
