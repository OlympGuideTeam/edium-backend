package live

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/repository"
)

func (s *Service) ConsumeWsToken(ctx context.Context, sessionID uuid.UUID, token string) (*repository.WsTokenPayload, error) {
	return s.liveTokens.ConsumeWsToken(ctx, sessionID, token)
}

func (s *Service) IsKicked(ctx context.Context, sessionID, attemptID uuid.UUID) (bool, error) {
	return s.liveParticip.IsKicked(ctx, sessionID, attemptID)
}

func (s *Service) GetSessionMeta(ctx context.Context, sessionID uuid.UUID) (*domain.LiveSessionMeta, error) {
	return s.liveSession.GetSessionMeta(ctx, sessionID)
}

func (s *Service) GetQuestions(ctx context.Context, sessionID uuid.UUID) ([]domain.QuestionWithOptions, error) {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if session == nil {
		return nil, nil
	}
	return s.quizzes.GetQuestionsWithOptions(ctx, session.QuizTemplateID)
}

func (s *Service) GetParticipants(ctx context.Context, sessionID uuid.UUID) ([]domain.LiveParticipant, error) {
	return s.liveParticip.GetParticipants(ctx, sessionID)
}

func (s *Service) SetParticipantStatus(ctx context.Context, sessionID, attemptID uuid.UUID, status string) error {
	return s.liveParticip.SetParticipantStatus(ctx, sessionID, attemptID, status)
}

func (s *Service) SetKicked(ctx context.Context, attemptID uuid.UUID) error {
	return s.attempts.SetKicked(ctx, attemptID)
}

func (s *Service) GetPhase(ctx context.Context, sessionID uuid.UUID) (domain.LivePhase, error) {
	return s.liveSession.GetPhase(ctx, sessionID)
}

func (s *Service) SetPhase(ctx context.Context, sessionID uuid.UUID, phase domain.LivePhase) error {
	return s.liveSession.SetPhase(ctx, sessionID, phase)
}

func (s *Service) SetCurrentQuestion(ctx context.Context, sessionID uuid.UUID, idx int, deadline time.Time) error {
	return s.liveSession.SetCurrentQuestion(ctx, sessionID, idx, deadline)
}

func (s *Service) SaveAnswer(ctx context.Context, sessionID, questionID, attemptID uuid.UUID, ans domain.LiveAnswer) error {
	return s.liveAnswers.SaveAnswer(ctx, sessionID, questionID, attemptID, ans)
}

func (s *Service) GetAnswer(ctx context.Context, sessionID, questionID, attemptID uuid.UUID) (*domain.LiveAnswer, error) {
	return s.liveAnswers.GetAnswer(ctx, sessionID, questionID, attemptID)
}

func (s *Service) GetAllAnswers(ctx context.Context, sessionID, questionID uuid.UUID) (map[uuid.UUID]domain.LiveAnswer, error) {
	return s.liveAnswers.GetAllAnswers(ctx, sessionID, questionID)
}

func (s *Service) GetAnsweredCount(ctx context.Context, sessionID, questionID uuid.UUID) (int, error) {
	return s.liveAnswers.GetAnsweredCount(ctx, sessionID, questionID)
}
