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
	ListSessionAttempts(ctx context.Context, sessionID, teacherID uuid.UUID) ([]domain.AttemptSummary, error)
	GetAttemptReview(ctx context.Context, attemptID uuid.UUID, callerID *uuid.UUID) (*domain.Attempt, []domain.AnswerWithQuestion, bool, error)
	GradeAttempt(ctx context.Context, attemptID, teacherID uuid.UUID, grades []domain.GradeItem) error
	PublishSession(ctx context.Context, sessionID, teacherID uuid.UUID) error
	GetUserStatistic(ctx context.Context, userID uuid.UUID) (*domain.UserStatistic, error)
}
