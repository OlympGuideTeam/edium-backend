package attempt

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/pkg/apperr"
)

func (s *Service) ListSessionAttempts(ctx context.Context, sessionID, teacherID uuid.UUID) ([]domain.AttemptSummary, error) {
	if err := s.requireSessionOwner(ctx, sessionID, teacherID); err != nil {
		return nil, err
	}
	attempts, err := s.attempts.FindBySessionID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("find attempts: %w", err)
	}
	return attempts, nil
}

func (s *Service) GradeSubmission(ctx context.Context, attemptID, submissionID, teacherID uuid.UUID, score float64, feedback *string) error {
	if score < 0 {
		return apperr.ErrScoreInvalid
	}
	return s.txManager.WithTx(ctx, func(ctx context.Context) error {
		attempt, err := s.attempts.GetByID(ctx, attemptID)
		if err != nil {
			return fmt.Errorf("get attempt: %w", err)
		}
		if attempt == nil {
			return apperr.ErrAttemptNotFound
		}
		if err := s.requireSessionOwner(ctx, attempt.SessionID, teacherID); err != nil {
			return err
		}
		submission, err := s.attempts.GetSubmissionByID(ctx, submissionID)
		if err != nil {
			return fmt.Errorf("get submission: %w", err)
		}
		if submission == nil || submission.AttemptID != attemptID {
			return apperr.ErrSubmissionNotFound
		}
		if _, err := s.attempts.InsertEvaluation(ctx, submissionID, domain.EvaluationStatusCompleted, &score, domain.FinalSourceTeacher, feedback); err != nil {
			return fmt.Errorf("insert evaluation: %w", err)
		}
		if err := s.attempts.EvaluateSubmission(ctx, submissionID, score, domain.FinalSourceTeacher, feedback); err != nil {
			return fmt.Errorf("evaluate submission: %w", err)
		}
		total, err := s.attempts.SumScores(ctx, attemptID)
		if err != nil {
			return fmt.Errorf("sum scores: %w", err)
		}
		if err := s.attempts.UpdateScore(ctx, attemptID, total); err != nil {
			return fmt.Errorf("update attempt score: %w", err)
		}
		return nil
	})
}

// GetAttemptReview возвращает попытку с ответами.
// Без JWT (callerID == nil) допустимо только при user_id попытки NULL в БД.
// Возвращает флаг enriched, indicating whether options/metadata were filled.
func (s *Service) GetAttemptReview(ctx context.Context, attemptID uuid.UUID, callerID *uuid.UUID) (*domain.Attempt, []domain.AnswerWithQuestion, bool, error) {
	attempt, err := s.attempts.GetByID(ctx, attemptID)
	if err != nil {
		return nil, nil, false, fmt.Errorf("get attempt: %w", err)
	}
	if attempt == nil {
		return nil, nil, false, apperr.ErrAttemptNotFound
	}

	var enrichTeacher bool

	if callerID == nil {
		if attempt.UserID != uuid.Nil {
			return nil, nil, false, apperr.ErrUnauthorized
		}
		if attempt.Status == domain.AttemptStatusInProgress {
			return nil, nil, false, apperr.ErrAttemptNotActive
		}
		enrichTeacher = false
	} else {
		caller := *callerID
		isOwner := attempt.UserID != uuid.Nil && attempt.UserID == caller
		if isOwner {
			if attempt.Status == domain.AttemptStatusInProgress {
				return nil, nil, false, apperr.ErrAttemptNotActive
			}
			enrichTeacher = false
		} else {
			if err := s.requireSessionOwner(ctx, attempt.SessionID, caller); err != nil {
				return nil, nil, false, err
			}
			enrichTeacher = true
		}
	}

	answers, err := s.attempts.GetAnswersWithQuestion(ctx, attemptID)
	if err != nil {
		return nil, nil, false, fmt.Errorf("get answers: %w", err)
	}

	if enrichTeacher {
		session, err := s.sessions.GetByID(ctx, attempt.SessionID)
		if err != nil {
			return nil, nil, false, fmt.Errorf("get session: %w", err)
		}
		questions, err := s.quizzes.GetQuestionsWithOptions(ctx, session.QuizTemplateID)
		if err != nil {
			return nil, nil, false, fmt.Errorf("get questions: %w", err)
		}
		qMap := make(map[uuid.UUID]domain.QuestionWithOptions, len(questions))
		for i := range questions {
			qMap[questions[i].ID] = questions[i]
		}
		for i := range answers {
			if q, ok := qMap[answers[i].QuestionID]; ok {
				answers[i].Options = q.Options
				answers[i].Metadata = q.Metadata
			}
		}
	}

	enriched := enrichTeacher || attempt.Status == domain.AttemptStatusCompleted
	return attempt, answers, enriched, nil
}

func (s *Service) CompleteAttempt(ctx context.Context, attemptID, teacherID uuid.UUID) error {
	return s.txManager.WithTx(ctx, func(ctx context.Context) error {
		attempt, err := s.attempts.GetByID(ctx, attemptID)
		if err != nil {
			return fmt.Errorf("get attempt: %w", err)
		}
		if attempt == nil {
			return apperr.ErrAttemptNotFound
		}
		if attempt.Status != domain.AttemptStatusGraded {
			return apperr.ErrAttemptNotGraded
		}
		if err := s.requireSessionOwner(ctx, attempt.SessionID, teacherID); err != nil {
			return err
		}
		session, err := s.sessions.GetByID(ctx, attempt.SessionID)
		if err != nil {
			return fmt.Errorf("get session: %w", err)
		}
		total, err := s.attempts.SumScores(ctx, attemptID)
		if err != nil {
			return fmt.Errorf("sum scores: %w", err)
		}
		grade := computeGrade(total, session.MaxScore)
		if err := s.attempts.CompleteGraded(ctx, attemptID, total, grade); err != nil {
			return fmt.Errorf("complete graded: %w", err)
		}
		attempt.Score = &total

		s.scheduleAttemptScored(ctx, attempt, total, float64(session.MaxScore))
		return nil
	})
}
