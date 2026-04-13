package quiz

import (
	"context"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

type quizService interface {
	CreateQuiz(ctx context.Context, authorID uuid.UUID, title string, description *string, settings domain.QuizDefaultSettings) (uuid.UUID, error)
}
