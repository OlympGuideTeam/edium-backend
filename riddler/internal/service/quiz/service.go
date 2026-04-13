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

func (s *Service) AddQuestion(ctx context.Context, quizID, authorID uuid.UUID, params domain.AddQuestionParams) (uuid.UUID, int, error) {
	quiz, err := s.quizzes.GetByID(ctx, quizID)
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return uuid.Nil, 0, apperr.ErrQuizNotFound
	}
	if quiz.AuthorID != authorID {
		return uuid.Nil, 0, apperr.ErrQuizForbidden
	}

	if err := validateQuestion(params.Type, params.Metadata, params.Options); err != nil {
		return uuid.Nil, 0, err
	}

	params.QuizTemplateID = quizID

	id, orderIndex, err := s.quizzes.AddQuestion(ctx, params)
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("add question: %w", err)
	}

	if params.Type == domain.QuestionTypeWithFreeAnswer && !quiz.NeedEvaluation {
		if err := s.quizzes.SetNeedEvaluation(ctx, quizID, true); err != nil {
			return uuid.Nil, 0, fmt.Errorf("set need_evaluation: %w", err)
		}
	}

	return id, orderIndex, nil
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

func validateQuestion(qType domain.QuestionType, metadata map[string]any, options []domain.AddOptionParams) error {
	switch qType {
	case domain.QuestionTypeSingleChoice:
		if len(options) < 2 {
			return apperr.ErrQuestionOptionsRequired
		}
		correct := 0
		for _, o := range options {
			if o.IsCorrect {
				correct++
			}
		}
		if correct != 1 {
			return apperr.ErrQuestionOneCorrect
		}

	case domain.QuestionTypeMultipleChoice:
		if len(options) < 2 {
			return apperr.ErrQuestionOptionsRequired
		}
		hasCorrect := false
		for _, o := range options {
			if o.IsCorrect {
				hasCorrect = true
				break
			}
		}
		if !hasCorrect {
			return apperr.ErrQuestionNoCorrect
		}

	case domain.QuestionTypeWithGivenAnswer:
		if !hasStringSlice(metadata, "correct_answers") {
			return apperr.ErrQuestionMetadataInvalid
		}

	case domain.QuestionTypeWithFreeAnswer:
		// метаданные не требуются

	case domain.QuestionTypeDrag:
		if !hasStringSlice(metadata, "correct_order") {
			return apperr.ErrQuestionMetadataInvalid
		}

	case domain.QuestionTypeConnection:
		if !hasStringSlice(metadata, "left") ||
			!hasStringSlice(metadata, "right") ||
			!hasStringMap(metadata, "correct_pairs") {
			return apperr.ErrQuestionMetadataInvalid
		}

	default:
		return apperr.ErrQuestionInvalidType
	}

	return nil
}

func hasStringSlice(metadata map[string]any, key string) bool {
	val, ok := metadata[key]
	if !ok {
		return false
	}
	slice, ok := val.([]any)
	if !ok || len(slice) == 0 {
		return false
	}
	for _, item := range slice {
		if _, ok := item.(string); !ok {
			return false
		}
	}
	return true
}

func hasStringMap(metadata map[string]any, key string) bool {
	val, ok := metadata[key]
	if !ok {
		return false
	}
	m, ok := val.(map[string]any)
	return ok && len(m) > 0
}
