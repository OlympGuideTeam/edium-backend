package quiz

import (
	"context"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

type quizService interface {
	CreateQuiz(ctx context.Context, authorID uuid.UUID, title string, description *string, settings domain.QuizDefaultSettings, attachToCourse *uuid.UUID) (uuid.UUID, error)
	AddQuestion(ctx context.Context, quizID, authorID uuid.UUID, params domain.AddQuestionParams) (uuid.UUID, int, error)
	UpdateQuiz(ctx context.Context, id, authorID uuid.UUID, title, description *string, settings *domain.QuizDefaultSettings) error
	ReorderQuestions(ctx context.Context, quizID, authorID uuid.UUID, questionIDs []uuid.UUID) error
	DeleteQuestion(ctx context.Context, quizID, questionID, authorID uuid.UUID) error
	PublishQuiz(ctx context.Context, id, authorID uuid.UUID) error
	GetQuiz(ctx context.Context, id, userID uuid.UUID) (*domain.QuizDetail, error)
	GetQuizForStudent(ctx context.Context, id uuid.UUID) (*domain.QuizStudentView, error)
	ListQuizzes(ctx context.Context, role domain.Role) ([]domain.QuizListItem, error)
	ListMyQuizzes(ctx context.Context, authorID uuid.UUID) ([]domain.QuizListItem, error)
	CopyQuiz(ctx context.Context, quizID, authorID uuid.UUID, attachToCourse *uuid.UUID) (uuid.UUID, error)
	CreateTestCourseSession(ctx context.Context, quizTemplateID, moduleID uuid.UUID, p domain.CreateTestCourseSessionParams) (uuid.UUID, error)
	CreateLiveCourseSession(ctx context.Context, quizTemplateID, moduleID uuid.UUID, p domain.CreateLiveCourseSessionParams) (uuid.UUID, error)
	GenerateQuestions(ctx context.Context, quizID, authorID uuid.UUID, text string) (uuid.UUID, error)
}
