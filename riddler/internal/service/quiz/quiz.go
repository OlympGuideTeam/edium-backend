package quiz

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/pkg/apperr"
)

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

func (s *Service) GetQuiz(ctx context.Context, id, authorID uuid.UUID) (*domain.QuizDetail, error) {
	quiz, err := s.quizzes.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return nil, apperr.ErrQuizNotFound
	}
	if quiz.AuthorID != authorID {
		return nil, apperr.ErrQuizForbidden
	}

	questions, err := s.quizzes.GetQuestionsWithOptions(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get questions: %w", err)
	}

	return &domain.QuizDetail{QuizTemplate: *quiz, Questions: questions}, nil
}

func (s *Service) UpdateQuiz(ctx context.Context, id, authorID uuid.UUID, title, description *string) error {
	if title != nil && strings.TrimSpace(*title) == "" {
		return apperr.ErrQuizEmptyTitle
	}

	quiz, err := s.quizzes.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return apperr.ErrQuizNotFound
	}
	if quiz.AuthorID != authorID {
		return apperr.ErrQuizForbidden
	}

	if err := s.quizzes.Update(ctx, id, title, description); err != nil {
		return fmt.Errorf("update quiz: %w", err)
	}
	return nil
}

func (s *Service) PublishQuiz(ctx context.Context, id, authorID uuid.UUID, isPublic bool) error {
	quiz, err := s.quizzes.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return apperr.ErrQuizNotFound
	}
	if quiz.AuthorID != authorID {
		return apperr.ErrQuizForbidden
	}
	if !quiz.IsDraft {
		return apperr.ErrQuizAlreadyPublished
	}

	return s.txManager.WithTx(ctx, func(ctx context.Context) error {
		if err := s.quizzes.Publish(ctx, id, isPublic); err != nil {
			return fmt.Errorf("publish quiz: %w", err)
		}

		if isPublic && !quiz.NeedEvaluation {
			totalTimeLimitSec := s.computeTotalTimeLimit(quiz.DefaultSettings, quiz.QuestionCount)

			if _, err := s.sessions.Create(ctx, domain.CreateSessionParams{
				QuizTemplateID:    id,
				Mode:              domain.SessionModeTest,
				Status:            domain.SessionStatusActive,
				TotalTimeLimitSec: &totalTimeLimitSec,
			}); err != nil {
				return fmt.Errorf("create session: %w", err)
			}
		}

		return nil
	})
}

func (s *Service) computeTotalTimeLimit(settings domain.QuizDefaultSettings, questionCount int) int {
	if settings.TotalTimeLimitSec != nil {
		return *settings.TotalTimeLimitSec
	}
	return 60 * questionCount
}

func (s *Service) CopyQuiz(ctx context.Context, quizID, authorID uuid.UUID) (uuid.UUID, error) {
	quiz, err := s.quizzes.GetByID(ctx, quizID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return uuid.Nil, apperr.ErrQuizNotFound
	}
	if quiz.IsDraft {
		return uuid.Nil, apperr.ErrQuizNotPublished
	}

	newID, err := s.quizzes.Copy(ctx, quizID, authorID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("copy quiz: %w", err)
	}
	return newID, nil
}

func (s *Service) ListQuizzes(ctx context.Context, role domain.Role) ([]domain.QuizListItem, error) {
	studentOnly := role == domain.RoleStudent
	items, err := s.quizzes.ListPublished(ctx, studentOnly)
	if err != nil {
		return nil, fmt.Errorf("list quizzes: %w", err)
	}
	return items, nil
}

func (s *Service) ListMyQuizzes(ctx context.Context, authorID uuid.UUID) ([]domain.QuizListItem, error) {
	items, err := s.quizzes.ListByAuthor(ctx, authorID)
	if err != nil {
		return nil, fmt.Errorf("list my quizzes: %w", err)
	}
	return items, nil
}
