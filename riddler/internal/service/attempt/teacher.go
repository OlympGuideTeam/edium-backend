package attempt

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/pkg/apperr"
)

func (s *Service) ListAwaitingReview(ctx context.Context, authorID uuid.UUID) ([]domain.AwaitingReviewSession, error) {
	return s.sessions.FindAwaitingReview(ctx, authorID)
}

func (s *Service) GetStudentDashboard(ctx context.Context, userID uuid.UUID) (*domain.StudentDashboard, error) {
	grades, err := s.sessions.FindStudentRecentGrades(ctx, userID, 5)
	if err != nil {
		return nil, fmt.Errorf("recent grades: %w", err)
	}
	active, err := s.sessions.FindStudentActiveTests(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("active tests: %w", err)
	}
	return &domain.StudentDashboard{
		RecentGrades: grades,
		ActiveTests:  active,
	}, nil
}

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

func (s *Service) GradeAttempt(ctx context.Context, attemptID, teacherID uuid.UUID, grades []domain.GradeItem) error {
	for _, g := range grades {
		if g.Score < 0 {
			return apperr.ErrScoreInvalid
		}
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
		for _, g := range grades {
			submission, err := s.attempts.GetSubmissionByID(ctx, g.SubmissionID)
			if err != nil {
				return fmt.Errorf("get submission: %w", err)
			}
			if submission == nil || submission.AttemptID != attemptID {
				return apperr.ErrSubmissionNotFound
			}
			if _, err := s.attempts.InsertEvaluation(ctx, g.SubmissionID, domain.EvaluationStatusCompleted, &g.Score, domain.FinalSourceTeacher, g.Feedback); err != nil {
				return fmt.Errorf("insert evaluation: %w", err)
			}
			if err := s.attempts.EvaluateSubmission(ctx, g.SubmissionID, g.Score, domain.FinalSourceTeacher, g.Feedback); err != nil {
				return fmt.Errorf("evaluate submission: %w", err)
			}
		}
		hasUngraded, err := s.attempts.HasUngradedFreeAnswers(ctx, attemptID)
		if err != nil {
			return fmt.Errorf("check ungraded: %w", err)
		}
		if hasUngraded {
			return apperr.ErrAttemptNotAllGraded
		}
		total, err := s.attempts.SumScores(ctx, attemptID)
		if err != nil {
			return fmt.Errorf("sum scores: %w", err)
		}
		session, err := s.sessions.GetByID(ctx, attempt.SessionID)
		if err != nil {
			return fmt.Errorf("get session: %w", err)
		}
		grade := computeGrade(total, session.MaxScore)
		return s.attempts.SetCompleted(ctx, attemptID, total, grade)
	})
}

func (s *Service) PublishSession(ctx context.Context, sessionID, teacherID uuid.UUID) error {
	if err := s.requireSessionOwner(ctx, sessionID, teacherID); err != nil {
		return err
	}
	return s.txManager.WithTx(ctx, func(ctx context.Context) error {
		attempts, err := s.attempts.FindBySessionID(ctx, sessionID)
		if err != nil {
			return fmt.Errorf("find attempts: %w", err)
		}
		for _, a := range attempts {
			if a.Status == domain.AttemptStatusKicked {
				continue
			}
			if a.Status != domain.AttemptStatusCompleted {
				return apperr.ErrAttemptNotCompleted
			}
		}
		session, err := s.sessions.GetByID(ctx, sessionID)
		if err != nil {
			return fmt.Errorf("get session: %w", err)
		}
		if err := s.attempts.BulkPublishBySessionID(ctx, sessionID); err != nil {
			return fmt.Errorf("bulk publish: %w", err)
		}
		for _, a := range attempts {
			if a.Status == domain.AttemptStatusKicked {
				continue
			}
			score := 0.0
			if a.Score != nil {
				score = *a.Score
			}
			s.scheduleAttemptScored(ctx, &domain.Attempt{
				ID:        a.ID,
				SessionID: sessionID,
				UserID:    a.UserID,
			}, score, float64(session.MaxScore), teacherID, domain.FinalSourceTeacher)
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

	session, err := s.sessions.GetByID(ctx, attempt.SessionID)
	if err != nil {
		return nil, nil, false, fmt.Errorf("get session: %w", err)
	}

	answers, err := s.attempts.GetAnswersWithQuestion(ctx, attemptID, session.QuizTemplateID)
	if err != nil {
		return nil, nil, false, fmt.Errorf("get answers: %w", err)
	}

	questions, err := s.quizzes.GetQuestionsWithOptions(ctx, session.QuizTemplateID)
	if err != nil {
		return nil, nil, false, fmt.Errorf("get questions: %w", err)
	}
	qMap := make(map[uuid.UUID]domain.QuestionWithOptions, len(questions))
	for i := range questions {
		qMap[questions[i].ID] = questions[i]
	}

	enriched := enrichTeacher || attempt.Status == domain.AttemptStatusPublished
	for i := range answers {
		if q, ok := qMap[answers[i].QuestionID]; ok {
			answers[i].Options = q.Options
			answers[i].Metadata = q.Metadata
		}
	}

	return attempt, answers, enriched, nil
}
