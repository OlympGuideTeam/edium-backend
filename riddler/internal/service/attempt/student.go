package attempt

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/pkg/apperr"
)

type attemptCreatedPayload struct {
	AttemptID uuid.UUID `json:"attempt_id"`
	SessionID uuid.UUID `json:"session_id"`
	UserID    uuid.UUID `json:"user_id"`
}

func (s *Service) Create(ctx context.Context, sessionID, userID uuid.UUID) (*domain.Attempt, []domain.QuestionForStudent, error) {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, nil, fmt.Errorf("get session: %w", err)
	}
	if session == nil {
		return nil, nil, apperr.ErrSessionNotFound
	}
	if err := validateSessionForAttempt(session); err != nil {
		return nil, nil, err
	}

	questions, err := s.quizzes.GetQuestionsWithOptions(ctx, session.QuizTemplateID)
	if err != nil {
		return nil, nil, fmt.Errorf("get questions: %w", err)
	}

	order := make([]uuid.UUID, len(questions))
	for i := range questions {
		order[i] = questions[i].ID
	}
	if session.ShuffleQuestions != nil && *session.ShuffleQuestions {
		rand.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
	}

	var attemptID uuid.UUID
	err = s.txManager.WithTx(ctx, func(ctx context.Context) error {
		var innerErr error
		attemptID, innerErr = s.attempts.Create(ctx, sessionID, userID, order)
		if innerErr != nil {
			return fmt.Errorf("create attempt: %w", innerErr)
		}
		payload, _ := json.Marshal(attemptCreatedPayload{AttemptID: attemptID, SessionID: sessionID, UserID: userID})
		if err := s.tasks.Schedule(ctx, domain.TaskTypeAttemptCreatedPublisher, payload); err != nil {
			return fmt.Errorf("schedule attempt.created: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	indexByID := make(map[uuid.UUID]domain.QuestionWithOptions, len(questions))
	for i := range questions {
		indexByID[questions[i].ID] = questions[i]
	}
	ordered := make([]domain.QuestionForStudent, len(order))
	for i, id := range order {
		ordered[i] = sanitizeQuestion(indexByID[id])
	}

	attempt := &domain.Attempt{
		ID:        attemptID,
		SessionID: sessionID,
		UserID:    userID,
		Status:    domain.AttemptStatusInProgress,
	}
	return attempt, ordered, nil
}

func (s *Service) SubmitAnswer(ctx context.Context, attemptID, userID, questionID uuid.UUID, answerData map[string]any) error {
	attempt, err := s.attempts.GetByID(ctx, attemptID)
	if err != nil {
		return fmt.Errorf("get attempt: %w", err)
	}
	if attempt == nil {
		return apperr.ErrAttemptNotFound
	}
	if attempt.UserID == uuid.Nil || attempt.UserID != userID {
		return apperr.ErrAttemptForbidden
	}
	if attempt.Status != domain.AttemptStatusInProgress {
		return apperr.ErrAttemptNotActive
	}

	if expired, err := s.isExpired(ctx, attempt); err != nil {
		return fmt.Errorf("check expiry: %w", err)
	} else if expired {
		if err := s.finishAttempt(ctx, attempt); err != nil {
			return fmt.Errorf("auto-finish expired: %w", err)
		}
		return apperr.ErrAttemptExpired
	}

	if _, err := s.attempts.UpsertAnswer(ctx, attemptID, questionID, answerData); err != nil {
		return fmt.Errorf("upsert answer: %w", err)
	}
	return nil
}

func (s *Service) Finish(ctx context.Context, attemptID, userID uuid.UUID) error {
	attempt, err := s.attempts.GetByID(ctx, attemptID)
	if err != nil {
		return fmt.Errorf("get attempt: %w", err)
	}
	if attempt == nil {
		return apperr.ErrAttemptNotFound
	}
	if attempt.UserID == uuid.Nil || attempt.UserID != userID {
		return apperr.ErrAttemptForbidden
	}
	if attempt.Status != domain.AttemptStatusInProgress {
		return apperr.ErrAttemptNotActive
	}

	return s.finishAttempt(ctx, attempt)
}

func validateSessionForAttempt(session *domain.QuizSession) error {
	switch session.Mode {
	case domain.SessionModeTest:
		switch session.ComputedStatus() {
		case domain.SessionStatusNotStarted:
			return apperr.ErrSessionNotStarted
		case domain.SessionStatusFinished:
			return apperr.ErrSessionDeadlinePassed
		}
	case domain.SessionModeLive:
		if session.Status != domain.SessionStatusWaiting {
			return apperr.ErrSessionNotActive
		}
	}
	return nil
}
