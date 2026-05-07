package live

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/pkg/apperr"
	"riddler/internal/repository"
)

type attemptCreatedPayload struct {
	AttemptID uuid.UUID `json:"attempt_id"`
	SessionID uuid.UUID `json:"session_id"`
	UserID    uuid.UUID `json:"user_id"`
}

func (s *Service) JoinLiveSession(ctx context.Context, sessionID uuid.UUID, userID *uuid.UUID, name *string) (attemptID uuid.UUID, wsToken string, err error) {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("get session: %w", err)
	}
	if session == nil || session.Mode != domain.SessionModeLive {
		return uuid.Nil, "", apperr.ErrSessionNotFound
	}
	if session.Status == domain.SessionStatusFinished {
		return uuid.Nil, "", apperr.ErrSessionCompleted
	}

	if session.Source == domain.LiveSourceCourse && userID == nil {
		return uuid.Nil, "", apperr.ErrUnauthorized
	}

	// реконнект: если у участника уже есть attempt — выдать новый ws_token без проверки фазы
	if userID != nil {
		existing, err := s.attempts.GetBySessionAndUser(ctx, sessionID, *userID)
		if err != nil {
			return uuid.Nil, "", fmt.Errorf("check existing attempt: %w", err)
		}
		if existing != nil {
			kicked, err := s.liveParticip.IsKicked(ctx, sessionID, existing.ID)
			if err != nil {
				return uuid.Nil, "", fmt.Errorf("check kicked: %w", err)
			}
			if kicked {
				return uuid.Nil, "", apperr.ErrLiveKicked
			}

			wsToken, err = s.liveTokens.IssueWsToken(ctx, sessionID, repository.WsTokenPayload{
				Role:      domain.RoleStudent,
				AttemptID: existing.ID.String(),
			})
			if err != nil {
				return uuid.Nil, "", fmt.Errorf("issue ws_token: %w", err)
			}
			return existing.ID, wsToken, nil
		}
	}

	// новый участник — только в фазе lobby
	phase, err := s.liveSession.GetPhase(ctx, sessionID)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("get phase: %w", err)
	}
	if phase != domain.LivePhaseLobby {
		return uuid.Nil, "", apperr.ErrLiveNotInLobby
	}

	if session.Source == domain.LiveSourceLibrary && (name == nil || *name == "") {
		return uuid.Nil, "", apperr.ErrBadRequest
	}

	attemptID, err = s.attempts.CreateLiveAttempt(ctx, sessionID, userID, name)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("create attempt: %w", err)
	}

	if session.Source == domain.LiveSourceCourse && userID != nil {
		payload, _ := json.Marshal(attemptCreatedPayload{AttemptID: attemptID, SessionID: sessionID, UserID: *userID})
		if err := s.tasks.Schedule(ctx, domain.TaskTypeAttemptCreatedPublisher, payload); err != nil {
			return uuid.Nil, "", fmt.Errorf("schedule attempt.created: %w", err)
		}
	}

	participant := domain.LiveParticipant{
		AttemptID: attemptID,
		UserID:    userID,
		Name:      name,
		Status:    "connected",
	}
	if err = s.liveParticip.AddParticipant(ctx, sessionID, participant); err != nil {
		return uuid.Nil, "", fmt.Errorf("add participant: %w", err)
	}

	wsToken, err = s.liveTokens.IssueWsToken(ctx, sessionID, repository.WsTokenPayload{
		Role:      domain.RoleStudent,
		AttemptID: attemptID.String(),
	})
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("issue ws_token: %w", err)
	}

	return attemptID, wsToken, nil
}
