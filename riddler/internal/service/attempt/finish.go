package attempt

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

func (s *Service) isExpired(ctx context.Context, attempt *domain.Attempt) (bool, error) {
	session, err := s.sessions.GetByID(ctx, attempt.SessionID)
	if err != nil {
		return false, fmt.Errorf("get session: %w", err)
	}
	if session == nil || session.TotalTimeLimitSec == nil {
		return false, nil
	}
	deadline := attempt.StartedAt.Add(time.Duration(*session.TotalTimeLimitSec) * time.Second)
	return time.Now().After(deadline), nil
}

func (s *Service) finishAttempt(ctx context.Context, attempt *domain.Attempt) error {
	return s.txManager.WithTx(ctx, func(ctx context.Context) error {
		answers, err := s.attempts.GetAnswers(ctx, attempt.ID)
		if err != nil {
			return fmt.Errorf("get answers: %w", err)
		}

		session, err := s.sessions.GetByID(ctx, attempt.SessionID)
		if err != nil {
			return fmt.Errorf("get session: %w", err)
		}

		questions, err := s.quizzes.GetQuestionsWithOptions(ctx, session.QuizTemplateID)
		if err != nil {
			return fmt.Errorf("get questions: %w", err)
		}
		questionMap := make(map[uuid.UUID]domain.QuestionWithOptions, len(questions))
		for i := range questions {
			questionMap[questions[i].ID] = questions[i]
		}

		var totalScore float64
		for _, ans := range answers {
			q, ok := questionMap[ans.QuestionID]
			if !ok {
				continue
			}
			score := gradeAnswer(q, ans.AnswerData)
			src := domain.FinalSourceAuto
			if err := s.attempts.EvaluateSubmission(ctx, ans.ID, score, src, nil); err != nil {
				return fmt.Errorf("evaluate submission: %w", err)
			}
			totalScore += score
		}

		return s.attempts.Complete(ctx, attempt.ID, totalScore)
	})
}
