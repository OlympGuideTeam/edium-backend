package quiz

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/pkg/apperr"
)

type Service struct {
	quizzes quizRepository
}

func NewService(quizzes quizRepository) *Service {
	return &Service{quizzes: quizzes}
}

func (s *Service) CreateQuiz(ctx context.Context, authorID uuid.UUID, title string, description *string, settings domain.QuizDefaultSettings) (uuid.UUID, error) {
	if strings.TrimSpace(title) == "" {
		return uuid.Nil, apperr.ErrQuizEmptyTitle
	}

	id, err := s.quizzes.Create(ctx, authorID, title, description, settings)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create quiz: %w", err)
	}

	return id, nil
}
