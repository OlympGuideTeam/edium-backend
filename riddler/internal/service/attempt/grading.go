package attempt

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

func (s *Service) ApplyLLMGrade(ctx context.Context, evalID uuid.UUID, score0to10 int, feedback *string) error {
	return s.txManager.WithTx(ctx, func(ctx context.Context) error {
		eval, err := s.attempts.GetEvaluationByID(ctx, evalID)
		if err != nil {
			return fmt.Errorf("get evaluation: %w", err)
		}
		if eval == nil {
			return fmt.Errorf("evaluation not found: %s", evalID)
		}

		question, err := s.questions.GetQuestionByID(ctx, eval.QuestionID)
		if err != nil {
			return fmt.Errorf("get question: %w", err)
		}

		finalScore := float64(score0to10) * float64(question.MaxScore) / 10.0

		if err := s.attempts.CompleteEvaluation(ctx, evalID, finalScore, feedback); err != nil {
			return fmt.Errorf("complete evaluation: %w", err)
		}
		if err := s.attempts.UpdateSubmissionFinalScore(ctx, eval.SubmissionID, finalScore, domain.FinalSourceLLM, feedback); err != nil {
			return fmt.Errorf("update submission: %w", err)
		}

		pending, err := s.attempts.HasPendingLLMEvaluations(ctx, eval.AttemptID)
		if err != nil {
			return fmt.Errorf("check pending: %w", err)
		}
		if pending {
			return nil
		}

		total, err := s.attempts.SumScores(ctx, eval.AttemptID)
		if err != nil {
			return fmt.Errorf("sum scores: %w", err)
		}
		if err := s.attempts.SetGraded(ctx, eval.AttemptID, total); err != nil {
			return fmt.Errorf("set graded: %w", err)
		}

		attempt, err := s.attempts.GetByID(ctx, eval.AttemptID)
		if err != nil {
			return fmt.Errorf("get attempt: %w", err)
		}

		session, err := s.sessions.GetByID(ctx, attempt.SessionID)
		if err != nil {
			return fmt.Errorf("get session: %w", err)
		}

		quiz, err := s.quizzes.GetByID(ctx, session.QuizTemplateID)
		if err != nil {
			return fmt.Errorf("get quiz: %w", err)
		}

		s.scheduleAttemptScored(ctx, attempt, total, float64(session.MaxScore), quiz.AuthorID, domain.FinalSourceLLM)
		return nil
	})
}
