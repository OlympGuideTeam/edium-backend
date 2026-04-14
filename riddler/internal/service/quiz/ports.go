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
	DeleteQuestion(ctx context.Context, quizID, questionID uuid.UUID) error
	Publish(ctx context.Context, id uuid.UUID, isPublic bool) error
	GetQuestionsWithOptions(ctx context.Context, quizID uuid.UUID) ([]domain.QuestionWithOptions, error)
	SetNeedEvaluation(ctx context.Context, quizID uuid.UUID, value bool) error
	HasFreeAnswerQuestions(ctx context.Context, quizID uuid.UUID) (bool, error)
	ListPublished(ctx context.Context, needEvaluationFalseOnly bool) ([]domain.QuizListItem, error)
	ListByAuthor(ctx context.Context, authorID uuid.UUID) ([]domain.QuizListItem, error)
	Copy(ctx context.Context, sourceID, newAuthorID uuid.UUID) (uuid.UUID, error)
}

type sessionService interface {
	Create(ctx context.Context, p domain.CreateSessionParams) (uuid.UUID, error)
	GetActiveTestSession(ctx context.Context, quizTemplateID uuid.UUID) (*domain.QuizSession, error)
}
