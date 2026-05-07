package attempt

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

func (s *Service) RunSessionAutoCloseWorker(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.autoCloseSessions(ctx); err != nil {
				slog.Error("session-auto-close-worker: ошибка", "err", err)
			}
		}
	}
}

func (s *Service) autoCloseSessions(ctx context.Context) error {
	sessions, err := s.sessionGrading.FindSessionsReadyToAutoClose(ctx)
	if err != nil {
		return err
	}
	for i := range sessions {
		if err := s.autoCloseSession(ctx, &sessions[i]); err != nil {
			slog.Error("session-auto-close-worker: ошибка сессии", "session_id", sessions[i].ID, "err", err)
		}
	}
	return nil
}

func (s *Service) autoCloseSession(ctx context.Context, session *domain.QuizSession) error {
	quiz, err := s.quizzes.GetByID(ctx, session.QuizTemplateID)
	if err != nil {
		return err
	}
	if quiz == nil {
		return nil
	}

	if quiz.NeedEvaluation {
		return s.sessionGrading.UpdateStatus(ctx, session.ID, domain.SessionStatusFinished)
	}

	return s.txManager.WithTx(ctx, func(ctx context.Context) error {
		attempts, err := s.attempts.FindBySessionID(ctx, session.ID)
		if err != nil {
			return err
		}
		if err := s.attempts.BulkPublishBySessionID(ctx, session.ID); err != nil {
			return err
		}
		if err := s.sessionGrading.UpdateStatus(ctx, session.ID, domain.SessionStatusFinished); err != nil {
			return err
		}
		for _, a := range attempts {
			if a.Status == domain.AttemptStatusKicked || a.UserID == uuid.Nil {
				continue
			}
			score := 0.0
			if a.Score != nil {
				score = *a.Score
			}
			s.scheduleAttemptScored(ctx, &domain.Attempt{
				ID:        a.ID,
				SessionID: session.ID,
				UserID:    a.UserID,
			}, score, float64(session.MaxScore), session.TeacherID, domain.FinalSourceAuto)
		}
		return nil
	})
}
