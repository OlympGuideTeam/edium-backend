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

func (s *Service) CreateLiveCourseSession(ctx context.Context, authorID, quizTemplateID, moduleID uuid.UUID, questionTimeLimitSec *int) (uuid.UUID, error) {
	quiz, err := s.quizzes.GetByID(ctx, quizTemplateID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return uuid.Nil, apperr.ErrQuizNotFound
	}
	if quiz.AuthorID != authorID {
		return uuid.Nil, apperr.ErrQuizForbidden
	}
	if quiz.Source != domain.QuizSourceCourse {
		return uuid.Nil, apperr.ErrQuizForbidden
	}
	if quiz.NeedEvaluation {
		return uuid.Nil, apperr.ErrLiveTemplateInvalid
	}

	timeLimitSec := resolveTimeLimitSec(questionTimeLimitSec, quiz.DefaultSettings.QuestionTimeLimitSec)

	var sessionID uuid.UUID
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		var innerErr error
		sessionID, innerErr = s.sessions.Create(ctx, domain.CreateSessionParams{
			QuizTemplateID:       quizTemplateID,
			Mode:                 domain.SessionModeLive,
			Source:               domain.LiveSourceCourse,
			MaxScore:             quiz.MaxScore,
			QuestionTimeLimitSec: &timeLimitSec,
			Status:               domain.SessionStatusNotStarted,
		})
		if innerErr != nil {
			return fmt.Errorf("create session: %w", innerErr)
		}

		qtID := quizTemplateID
		payload, _ := json.Marshal(domain.CourseSessionCreatedPayload{
			SessionID:            sessionID,
			ModuleID:             moduleID,
			QuizTemplateID:       &qtID,
			CourseID:             quiz.CourseID,
			Title:                quiz.Title,
			Mode:                 string(domain.SessionModeLive),
			QuestionTimeLimitSec: &timeLimitSec,
		})
		return s.tasks.Schedule(ctx, domain.TaskTypeCourseSessionCreatedPublisher, payload)
	})
	if err != nil {
		return uuid.Nil, err
	}

	return sessionID, nil
}

func (s *Service) CreateLiveLibrarySession(ctx context.Context, authorID, quizTemplateID uuid.UUID, questionTimeLimitSec *int) (uuid.UUID, error) {
	quiz, err := s.quizzes.GetByID(ctx, quizTemplateID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return uuid.Nil, apperr.ErrQuizNotFound
	}
	if quiz.Source != domain.QuizSourceLibrary || !quiz.IsPublic {
		return uuid.Nil, apperr.ErrLiveNotLibrary
	}
	if quiz.NeedEvaluation {
		return uuid.Nil, apperr.ErrLiveTemplateInvalid
	}

	timeLimitSec := resolveTimeLimitSec(questionTimeLimitSec, quiz.DefaultSettings.QuestionTimeLimitSec)

	hostID := authorID
	return s.sessions.Create(ctx, domain.CreateSessionParams{
		QuizTemplateID:       quizTemplateID,
		Mode:                 domain.SessionModeLive,
		Source:               domain.LiveSourceLibrary,
		LiveHostUserID:       &hostID,
		MaxScore:             quiz.MaxScore,
		QuestionTimeLimitSec: &timeLimitSec,
		Status:               domain.SessionStatusNotStarted,
	})
}

func (s *Service) StartLiveSession(ctx context.Context, sessionID, callerID uuid.UUID) (wsToken, joinCode string, err error) {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return "", "", fmt.Errorf("get session: %w", err)
	}
	if session == nil || session.Mode != domain.SessionModeLive {
		return "", "", apperr.ErrSessionNotFound
	}

	quiz, err := s.quizzes.GetByID(ctx, session.QuizTemplateID)
	if err != nil {
		return "", "", fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return "", "", apperr.ErrQuizNotFound
	}

	switch session.Source {
	case domain.LiveSourceCourse:
		if quiz.AuthorID != callerID {
			return "", "", apperr.ErrQuizForbidden
		}
	case domain.LiveSourceLibrary:
		if session.LiveHostUserID != nil {
			if *session.LiveHostUserID != callerID {
				return "", "", apperr.ErrQuizForbidden
			}
		} else if quiz.AuthorID != callerID {
			// Сессии до миграции live_host_user_id создавал только автор шаблона.
			return "", "", apperr.ErrQuizForbidden
		}
	}

	joinCode, err = s.liveSession.InitSession(ctx, sessionID, quiz, *session.QuestionTimeLimitSec, session.Source, callerID)
	if err != nil {
		return "", "", fmt.Errorf("init session: %w", err)
	}

	if err := s.sessions.UpdateStatus(ctx, sessionID, domain.SessionStatusRunning); err != nil {
		return "", "", fmt.Errorf("update session status: %w", err)
	}

	wsToken, err = s.liveTokens.IssueWsToken(ctx, sessionID, repository.WsTokenPayload{
		Role:   domain.RoleTeacher,
		UserID: callerID.String(),
	})
	if err != nil {
		return "", "", fmt.Errorf("issue ws_token: %w", err)
	}

	return wsToken, joinCode, nil
}

func (s *Service) ResolveLiveCode(ctx context.Context, code string) (*domain.LiveSessionMeta, error) {
	sessionID, err := s.liveSession.ResolveCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("resolve code: %w", err)
	}
	if sessionID == uuid.Nil {
		return nil, apperr.ErrLiveCodeExpired
	}

	meta, err := s.liveSession.GetSessionMeta(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session meta: %w", err)
	}
	if meta == nil {
		return nil, apperr.ErrLiveCodeExpired
	}

	return meta, nil
}

func (s *Service) ListLibraryLiveSessions(ctx context.Context, authorID uuid.UUID) ([]domain.LibraryLiveSession, error) {
	sessions, err := s.sessions.ListLiveLibrarySessions(ctx, authorID)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	for i, sess := range sessions {
		if sess.Status == domain.SessionStatusFinished {
			sessions[i].Phase = domain.LivePhaseCompleted
			continue
		}
		meta, err := s.liveSession.GetSessionMeta(ctx, sess.SessionID)
		if err != nil || meta == nil {
			sessions[i].Phase = domain.LivePhasePending
			continue
		}
		phase := meta.Phase
		if phase == "" {
			phase = domain.LivePhaseLobby
		}
		sessions[i].Phase = phase
		sessions[i].JoinCode = meta.JoinCode
		sessions[i].ParticipantsCount = meta.ParticipantsCount
	}
	return sessions, nil
}

func resolveTimeLimitSec(override, fromQuiz *int) int {
	if override != nil {
		return *override
	}
	if fromQuiz != nil {
		return *fromQuiz
	}
	return defaultQuestionTimeLimitSec
}
