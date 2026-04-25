package quiz

import (
	"context"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

type quizRepository interface {
	Create(ctx context.Context, authorID uuid.UUID, title string, description *string, settings domain.QuizDefaultSettings, source domain.QuizSource, courseID *uuid.UUID) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.QuizTemplate, error)
	AddQuestion(ctx context.Context, params domain.AddQuestionParams) (uuid.UUID, int, error)
	Update(ctx context.Context, id uuid.UUID, title, description *string, settings *domain.QuizDefaultSettings) error
	ReorderQuestions(ctx context.Context, quizID uuid.UUID, questionIDs []uuid.UUID) error
	DeleteQuestion(ctx context.Context, quizID, questionID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	Publish(ctx context.Context, id uuid.UUID) error
	SetLibrarySession(ctx context.Context, quizID, sessionID uuid.UUID) error
	GetQuestionsWithOptions(ctx context.Context, quizID uuid.UUID) ([]domain.QuestionWithOptions, error)
	SetNeedEvaluation(ctx context.Context, quizID uuid.UUID, value bool) error
	HasFreeAnswerQuestions(ctx context.Context, quizID uuid.UUID) (bool, error)
	ListPublished(ctx context.Context, needEvaluationFalseOnly bool) ([]domain.QuizListItem, error)
	ListByAuthor(ctx context.Context, authorID uuid.UUID) ([]domain.QuizListItem, error)
	Copy(ctx context.Context, sourceID, newAuthorID uuid.UUID, source domain.QuizSource, courseID *uuid.UUID) (uuid.UUID, error)
	SumMaxScore(ctx context.Context, quizTemplateID uuid.UUID) (int, error)
}

type sessionService interface {
	Create(ctx context.Context, p domain.CreateSessionParams) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.QuizSession, error)
	HasAttempts(ctx context.Context, sessionID uuid.UUID) (bool, error)
	HasSessions(ctx context.Context, quizTemplateID uuid.UUID) (bool, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type taskScheduler interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
}
