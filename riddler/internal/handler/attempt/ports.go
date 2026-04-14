package attempt

import (
	"context"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

type attemptService interface {
	Create(ctx context.Context, sessionID, userID uuid.UUID) (*domain.Attempt, []domain.QuestionForStudent, error)
	SubmitAnswer(ctx context.Context, attemptID, userID, questionID uuid.UUID, answerData map[string]any) error
	Finish(ctx context.Context, attemptID, userID uuid.UUID) error
	GetResult(ctx context.Context, attemptID, userID uuid.UUID) (*domain.AttemptResult, error)
}
