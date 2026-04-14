package attempt

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/infra/db"
	"riddler/internal/pkg/apperr"
)

type Service struct {
	attempts  attemptRepository
	sessions  sessionReader
	quizzes   quizReader
	txManager *db.TxManager
}

func NewService(attempts attemptRepository, sessions sessionReader, quizzes quizReader, txManager *db.TxManager) *Service {
	return &Service{attempts: attempts, sessions: sessions, quizzes: quizzes, txManager: txManager}
}

func (s *Service) Create(ctx context.Context, sessionID, userID uuid.UUID) (*domain.Attempt, []domain.QuestionForStudent, error) {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, nil, fmt.Errorf("get session: %w", err)
	}
	if session == nil {
		return nil, nil, apperr.ErrSessionNotFound
	}
	if session.Status != domain.SessionStatusActive {
		return nil, nil, apperr.ErrSessionNotActive
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

	attemptID, err := s.attempts.Create(ctx, sessionID, userID, order)
	if err != nil {
		return nil, nil, fmt.Errorf("create attempt: %w", err)
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
	if attempt.UserID != userID {
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
	if attempt.UserID != userID {
		return apperr.ErrAttemptForbidden
	}
	if attempt.Status != domain.AttemptStatusInProgress {
		return apperr.ErrAttemptNotActive
	}

	return s.finishAttempt(ctx, attempt)
}

func (s *Service) GetResult(ctx context.Context, attemptID, userID uuid.UUID) (*domain.AttemptResult, error) {
	attempt, err := s.attempts.GetByID(ctx, attemptID)
	if err != nil {
		return nil, fmt.Errorf("get attempt: %w", err)
	}
	if attempt == nil {
		return nil, apperr.ErrAttemptNotFound
	}
	if attempt.UserID != userID {
		return nil, apperr.ErrAttemptForbidden
	}
	if attempt.Status != domain.AttemptStatusCompleted {
		return nil, apperr.ErrAttemptNotActive
	}

	answers, err := s.attempts.GetAnswers(ctx, attemptID)
	if err != nil {
		return nil, fmt.Errorf("get answers: %w", err)
	}

	return &domain.AttemptResult{Attempt: *attempt, Answers: answers}, nil
}
