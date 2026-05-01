package live

import (
	"context"
	"fmt"

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
		if err := s.attempts.CompleteLive(ctx, p.AttemptID, totalScore); err != nil {
			return fmt.Errorf("complete attempt: %w", err)
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

func (s *Service) collectSubmissions(ctx context.Context, sessionID, attemptID uuid.UUID, questions []domain.QuestionWithOptions) ([]repository.BulkSubmission, float64, error) {
	submissions := make([]repository.BulkSubmission, 0, len(questions))
	var totalScore float64

	for _, q := range questions {
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
