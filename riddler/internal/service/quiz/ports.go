package quiz

import (
	"context"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

type quizRepository interface {
	Create(ctx context.Context, authorID uuid.UUID, title string, description *string, settings domain.QuizDefaultSettings) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.QuizTemplate, error)
	AddQuestion(ctx context.Context, params domain.AddQuestionParams) (uuid.UUID, int, error)
	Update(ctx context.Context, id uuid.UUID, title, description *string) error
	ReorderQuestions(ctx context.Context, quizID uuid.UUID, questionIDs []uuid.UUID) error
	SetNeedEvaluation(ctx context.Context, quizID uuid.UUID, value bool) error
	HasFreeAnswerQuestions(ctx context.Context, quizID uuid.UUID) (bool, error)
}
