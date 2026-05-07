package test

import (
	"context"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

type quizRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.QuizTemplate, error)
}

type sessionRepository interface {
	Create(ctx context.Context, p domain.CreateSessionParams) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.QuizSession, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.SessionStatus) error
}

type taskScheduler interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
}

type txManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type attemptFinisher interface {
	FinishInProgressBySession(ctx context.Context, sessionID uuid.UUID) error
}
