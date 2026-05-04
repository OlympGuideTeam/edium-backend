package attempt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/infra/db"
	"riddler/internal/pkg/apperr"
)

type Service struct {
	attempts       attemptRepository
	sessions       sessionReader
	sessionGrading sessionGradingRepo
	quizzes        quizReader
	questions      questionReader
	tasks          taskScheduler
	txManager      *db.TxManager
}

func NewService(attempts attemptRepository, sessions sessionReader, sessionGrading sessionGradingRepo, quizzes quizReader, questions questionReader, txManager *db.TxManager, tasks taskScheduler) *Service {
	return &Service{attempts: attempts, sessions: sessions, sessionGrading: sessionGrading, quizzes: quizzes, questions: questions, tasks: tasks, txManager: txManager}
}

func (s *Service) requireSessionOwner(ctx context.Context, sessionID, teacherID uuid.UUID) error {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}
	if session == nil {
		return apperr.ErrSessionNotFound
	}
	quiz, err := s.quizzes.GetByID(ctx, session.QuizTemplateID)
	if err != nil {
		return fmt.Errorf("get quiz: %w", err)
	}
	if quiz.AuthorID != teacherID {
		return apperr.ErrQuizForbidden
	}
	return nil
}

func (s *Service) GetUserStatistic(ctx context.Context, userID uuid.UUID) (*domain.UserStatistic, error) {
	return s.attempts.GetUserStatistic(ctx, userID)
}

func (s *Service) scheduleAttemptScored(ctx context.Context, attempt *domain.Attempt, totalScore, maxScore float64) {
	type payload struct {
		AttemptID  uuid.UUID `json:"attempt_id"`
		SessionID  uuid.UUID `json:"session_id"`
		UserID     uuid.UUID `json:"user_id,omitempty"`
		TotalScore float64   `json:"total_score"`
		MaxScore   float64   `json:"max_score"`
	}
	pl := payload{
		AttemptID:  attempt.ID,
		SessionID:  attempt.SessionID,
		TotalScore: totalScore,
		MaxScore:   maxScore,
	}
	if attempt.UserID != uuid.Nil {
		pl.UserID = attempt.UserID
	}
	data, _ := json.Marshal(pl)
	if err := s.tasks.Schedule(ctx, domain.TaskTypeAttemptScoredPublisher, data); err != nil {
		slog.ErrorContext(ctx, "schedule attempt.scored", "attempt_id", attempt.ID, "err", err)
	}
}
