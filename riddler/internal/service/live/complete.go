package live

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"riddler/internal/pkg/grading"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/repository"
)

func (s *Service) CompleteLiveSession(ctx context.Context, sessionID uuid.UUID) error {
	meta, err := s.liveSession.GetSessionMeta(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get session meta: %w", err)
	}
	if meta == nil {
		return nil
	}

	questions, err := s.quizzes.GetQuestionsWithOptions(ctx, meta.QuizTemplateID)
	if err != nil {
		return fmt.Errorf("get questions: %w", err)
	}

	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	participants, err := s.liveParticip.GetParticipants(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get participants: %w", err)
	}

	for _, p := range participants {
		if p.Status == "kicked" {
			if err := s.attempts.SetKicked(ctx, p.AttemptID); err != nil {
				return fmt.Errorf("set kicked: %w", err)
			}
			continue
		}

		submissions, totalScore, err := s.collectSubmissions(ctx, sessionID, p.AttemptID, questions)
		if err != nil {
			return err
		}

		if err := s.attempts.BulkInsertSubmissions(ctx, submissions); err != nil {
			return fmt.Errorf("bulk insert submissions: %w", err)
		}
		grade := grading.ComputeGrade(totalScore, session.MaxScore)
		if err := s.attempts.PublishLive(ctx, p.AttemptID, totalScore, grade); err != nil {
			return fmt.Errorf("publish attempt: %w", err)
		}
		if meta.Source == domain.LiveSourceCourse && p.UserID != nil {
			s.scheduleAttemptScored(ctx, p.AttemptID, sessionID, *p.UserID, totalScore, float64(session.MaxScore))
		}
	}

	if err := s.sessions.UpdateStatus(ctx, sessionID, domain.SessionStatusFinished); err != nil {
		return fmt.Errorf("update session status: %w", err)
	}

	code := ""
	if meta.JoinCode != nil {
		code = *meta.JoinCode
	}
	return s.liveSession.DeleteAll(ctx, sessionID, code)
}

func (s *Service) scheduleAttemptScored(ctx context.Context, attemptID, sessionID, userID uuid.UUID, totalScore, maxScore float64) {
	type payload struct {
		AttemptID  uuid.UUID          `json:"attempt_id"`
		SessionID  uuid.UUID          `json:"session_id"`
		UserID     uuid.UUID          `json:"user_id"`
		TotalScore float64            `json:"total_score"`
		MaxScore   float64            `json:"max_score"`
		GradedBy   domain.FinalSource `json:"graded_by"`
	}
	data, _ := json.Marshal(payload{
		AttemptID:  attemptID,
		SessionID:  sessionID,
		UserID:     userID,
		TotalScore: totalScore,
		MaxScore:   maxScore,
		GradedBy:   domain.FinalSourceAuto,
	})
	if err := s.tasks.Schedule(ctx, domain.TaskTypeAttemptScoredPublisher, data); err != nil {
		slog.ErrorContext(ctx, "live: schedule attempt.scored", "attempt_id", attemptID, "err", err)
	}
}

func (s *Service) collectSubmissions(ctx context.Context, sessionID, attemptID uuid.UUID, questions []domain.QuestionWithOptions) ([]repository.BulkSubmission, float64, error) {
	submissions := make([]repository.BulkSubmission, 0, len(questions))
	var totalScore float64

	for i := range questions {
		q := &questions[i]
		ans, err := s.liveAnswers.GetAnswer(ctx, sessionID, q.ID, attemptID)
		if err != nil {
			return nil, 0, fmt.Errorf("get answer: %w", err)
		}
		if ans == nil {
			continue
		}
		totalScore += ans.Score
		submissions = append(submissions, repository.BulkSubmission{
			AttemptID:   attemptID,
			QuestionID:  q.ID,
			AnswerData:  ans.AnswerData,
			FinalScore:  ans.Score,
			TimeTakenMs: ans.TimeTakenMs,
		})
	}

	return submissions, totalScore, nil
}
