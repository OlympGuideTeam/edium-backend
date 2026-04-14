package attempt

import (
	"context"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

type attemptRepository interface {
	Create(ctx context.Context, sessionID, userID uuid.UUID, questionOrder []uuid.UUID) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Attempt, error)
	UpsertAnswer(ctx context.Context, attemptID, questionID uuid.UUID, answerData map[string]any) (uuid.UUID, error)
	GetAnswers(ctx context.Context, attemptID uuid.UUID) ([]domain.AnswerSubmission, error)
	EvaluateSubmission(ctx context.Context, submissionID uuid.UUID, score float64, source domain.FinalSource, feedback *string) error
	Complete(ctx context.Context, attemptID uuid.UUID, score float64) error
	FindExpiredInProgress(ctx context.Context) ([]domain.Attempt, error)
}

type sessionReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.QuizSession, error)
}

type quizReader interface {
	GetQuestionsWithOptions(ctx context.Context, quizID uuid.UUID) ([]domain.QuestionWithOptions, error)
}
