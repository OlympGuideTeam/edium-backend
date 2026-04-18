package attempt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"riddler/internal/domain"
)

type gradingAnswer struct {
	EvalID string `json:"eval_id"`
	Text   string `json:"text"`
}

type gradingRequestPayload struct {
	QuestionText string          `json:"question_text"`
	Answers      []gradingAnswer `json:"answers"`
}

func (s *Service) RunSessionGradingWorker(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.dispatchPendingGradingSessions(ctx); err != nil {
				slog.Error("session-grading-worker: ошибка", "err", err)
			}
		}
	}
}

func (s *Service) dispatchPendingGradingSessions(ctx context.Context) error {
	sessions, err := s.sessionGrading.FindFinishedNeedingGrading(ctx)
	if err != nil {
		return fmt.Errorf("find sessions: %w", err)
	}
	for i := range sessions {
		if err := s.dispatchSession(ctx, &sessions[i]); err != nil {
			slog.Error("session-grading-worker: ошибка сессии", "session_id", sessions[i].ID, "err", err)
		}
	}
	return nil
}

func (s *Service) dispatchSession(ctx context.Context, session *domain.QuizSession) error {
	return s.txManager.WithTx(ctx, func(ctx context.Context) error {
		submissions, err := s.sessionGrading.GetFreeAnswerSubmissionsForSession(ctx, session.ID)
		if err != nil {
			return fmt.Errorf("get submissions: %w", err)
		}

		byQuestion := make(map[string]*gradingRequestPayload)
		questionOrder := make([]string, 0)
		for _, sub := range submissions {
			evalID, err := s.attempts.InsertEvaluation(ctx, sub.SubmissionID, domain.EvaluationStatusPending, nil, domain.FinalSourceLLM, nil)
			if err != nil {
				return fmt.Errorf("insert pending evaluation: %w", err)
			}
			qKey := sub.QuestionID.String()
			if _, exists := byQuestion[qKey]; !exists {
				byQuestion[qKey] = &gradingRequestPayload{QuestionText: sub.QuestionText}
				questionOrder = append(questionOrder, qKey)
			}
			byQuestion[qKey].Answers = append(byQuestion[qKey].Answers, gradingAnswer{
				EvalID: evalID.String(),
				Text:   sub.AnswerText,
			})
		}

		for _, qKey := range questionOrder {
			payload, _ := json.Marshal(byQuestion[qKey])
			if err := s.tasks.Schedule(ctx, domain.TaskTypeGradingRequestedPublisher, payload); err != nil {
				return fmt.Errorf("schedule grading task: %w", err)
			}
		}

		return s.sessionGrading.SetGradingSent(ctx, session.ID)
	})
}
