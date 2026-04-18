package quiz

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/pkg/apperr"
)

type generationRequestedPayload struct {
	JobID  uuid.UUID `json:"job_id"`
	QuizID uuid.UUID `json:"quiz_id"`
	Text   string    `json:"text"`
}

func (s *Service) GenerateQuestions(ctx context.Context, quizID, authorID uuid.UUID, text string) (uuid.UUID, error) {
	quiz, err := s.quizzes.GetByID(ctx, quizID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return uuid.Nil, apperr.ErrQuizNotFound
	}
	if quiz.AuthorID != authorID {
		return uuid.Nil, apperr.ErrQuizForbidden
	}

	jobID := uuid.New()
	payload, _ := json.Marshal(generationRequestedPayload{
		JobID:  jobID,
		QuizID: quizID,
		Text:   text,
	})
	if err := s.tasks.Schedule(ctx, domain.TaskTypeGenerationRequestedPublisher, payload); err != nil {
		return uuid.Nil, fmt.Errorf("schedule generation_requested: %w", err)
	}
	return jobID, nil
}

func (s *Service) AddGeneratedQuestions(ctx context.Context, quizID uuid.UUID, questions []domain.AddQuestionParams) error {
	quiz, err := s.quizzes.GetByID(ctx, quizID)
	if err != nil {
		return fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return apperr.ErrQuizNotFound
	}

	hasFreeAnswer := false
	return s.txManager.WithTx(ctx, func(ctx context.Context) error {
		for i := range questions {
			questions[i].QuizTemplateID = quizID
			if _, _, err := s.quizzes.AddQuestion(ctx, questions[i]); err != nil {
				return fmt.Errorf("add question: %w", err)
			}
			if questions[i].Type == domain.QuestionTypeWithFreeAnswer {
				hasFreeAnswer = true
			}
		}
		if hasFreeAnswer && !quiz.NeedEvaluation {
			if err := s.quizzes.SetNeedEvaluation(ctx, quizID, true); err != nil {
				return fmt.Errorf("set need_evaluation: %w", err)
			}
		}
		return nil
	})
}
