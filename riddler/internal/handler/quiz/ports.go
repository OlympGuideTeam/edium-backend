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
	DeleteQuiz(ctx context.Context, id, authorID uuid.UUID) error
	ReorderQuestions(ctx context.Context, quizID, authorID uuid.UUID, questionIDs []uuid.UUID) error
	DeleteQuestion(ctx context.Context, quizID, questionID, authorID uuid.UUID) error
	PublishQuiz(ctx context.Context, id, authorID uuid.UUID) error
	GetQuiz(ctx context.Context, id, userID uuid.UUID) (*domain.QuizDetail, error)
	GetQuizForStudent(ctx context.Context, id, userID uuid.UUID) (*domain.QuizStudentView, error)
	ListQuizzes(ctx context.Context, role domain.Role, userID *uuid.UUID, search *string) ([]domain.QuizListItem, error)
	ListMyQuizzes(ctx context.Context, authorID uuid.UUID, search *string) ([]domain.QuizListItem, error)
	CopyQuiz(ctx context.Context, quizID, authorID uuid.UUID, attachToCourse *uuid.UUID) (uuid.UUID, error)
	CreateTestCourseSessionInline(ctx context.Context, authorID uuid.UUID, p domain.CreateTestCourseSessionInlineParams) (uuid.UUID, uuid.UUID, error)
	CreateLiveCourseSessionInline(ctx context.Context, authorID uuid.UUID, p domain.CreateLiveCourseSessionInlineParams) (uuid.UUID, uuid.UUID, error)
	DeleteCourseSession(ctx context.Context, sessionID, authorID uuid.UUID) error
	GenerateQuestions(ctx context.Context, quizID, authorID uuid.UUID, text string) (uuid.UUID, error)
}
