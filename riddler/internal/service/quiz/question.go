package quiz

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/pkg/apperr"
)

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

	var id uuid.UUID
	var orderIndex int
	err = s.txManager.WithTx(ctx, func(ctx context.Context) error {
		var innerErr error
		id, orderIndex, innerErr = s.quizzes.AddQuestion(ctx, params)
		if innerErr != nil {
			return fmt.Errorf("add question: %w", innerErr)
		}
		if params.Type == domain.QuestionTypeWithFreeAnswer && !quiz.NeedEvaluation {
			if innerErr = s.quizzes.SetNeedEvaluation(ctx, quizID, true); innerErr != nil {
				return fmt.Errorf("set need_evaluation: %w", innerErr)
			}
		}
		return nil
	})
	if err != nil {
		return uuid.Nil, 0, err
	}

	return id, orderIndex, nil
}

func (s *Service) DeleteQuestion(ctx context.Context, quizID, questionID, authorID uuid.UUID) error {
	quiz, err := s.quizzes.GetByID(ctx, quizID)
	if err != nil {
		return fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return apperr.ErrQuizNotFound
	}
	if quiz.AuthorID != authorID {
		return apperr.ErrQuizForbidden
	}

	if err := s.quizzes.DeleteQuestion(ctx, quizID, questionID); err != nil {
		return fmt.Errorf("delete question: %w", err)
	}

	if quiz.NeedEvaluation {
		has, err := s.quizzes.HasFreeAnswerQuestions(ctx, quizID)
		if err != nil {
			return fmt.Errorf("check free_answer questions: %w", err)
		}
		if !has {
			if err := s.quizzes.SetNeedEvaluation(ctx, quizID, false); err != nil {
				return fmt.Errorf("set need_evaluation: %w", err)
			}
		}
	}

	return nil
}

func (s *Service) ReorderQuestions(ctx context.Context, quizID, authorID uuid.UUID, questionIDs []uuid.UUID) error {
	quiz, err := s.quizzes.GetByID(ctx, quizID)
	if err != nil {
		return fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return apperr.ErrQuizNotFound
	}
	if quiz.AuthorID != authorID {
		return apperr.ErrQuizForbidden
	}

	if err := s.quizzes.ReorderQuestions(ctx, quizID, questionIDs); err != nil {
		return fmt.Errorf("reorder questions: %w", err)
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
