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
			TeacherID:            authorID,
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

	return s.sessions.Create(ctx, domain.CreateSessionParams{
		QuizTemplateID:       quizTemplateID,
		Mode:                 domain.SessionModeLive,
		Source:               domain.LiveSourceLibrary,
		TeacherID:            authorID,
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
	if session.Status == domain.SessionStatusFinished {
		return "", "", apperr.ErrSessionCompleted
	}

	if session.TeacherID != callerID {
		return "", "", apperr.ErrQuizForbidden
	}

	quiz, err := s.quizzes.GetByID(ctx, session.QuizTemplateID)
	if err != nil {
		return "", "", fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return "", "", apperr.ErrQuizNotFound
	}

	currentPhase, phaseErr := s.liveSession.GetPhase(ctx, sessionID)
	switch {
	case phaseErr != nil || currentPhase == "" || currentPhase == domain.LivePhasePending:
		// Сессия ещё не инициализирована — запускаем впервые.
		joinCode, err = s.liveSession.InitSession(ctx, sessionID, quiz, *session.QuestionTimeLimitSec, session.Source, callerID)
		if err != nil {
			return "", "", fmt.Errorf("init session: %w", err)
		}
		if err := s.sessions.UpdateStatus(ctx, sessionID, domain.SessionStatusRunning); err != nil {
			return "", "", fmt.Errorf("update session status: %w", err)
		}
		if session.Source == domain.LiveSourceCourse && quiz.CourseID != nil && s.notifier != nil {
			s.notifier.NotifyLobbyOpened(sessionID, *quiz.CourseID, quiz.Title, *session.QuestionTimeLimitSec)
		}
	case currentPhase == domain.LivePhaseCompleted:
		return "", "", apperr.ErrSessionCompleted
	default:
		// Сессия уже запущена — переиздаём ws_token, join_code не меняем.
		meta, err := s.liveSession.GetSessionMeta(ctx, sessionID)
		if err != nil {
			return "", "", fmt.Errorf("get session meta: %w", err)
		}
		if meta == nil || meta.JoinCode == nil {
			return "", "", apperr.ErrSessionNotFound
		}
		joinCode = *meta.JoinCode
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

func (s *Service) GetActiveCourseLiveSessions(ctx context.Context, courseIDs []uuid.UUID) ([]domain.LiveLobbySnapshot, error) {
	sessions, err := s.sessions.FindRunningCourseLiveSessions(ctx, courseIDs)
	if err != nil {
		return nil, fmt.Errorf("find running sessions: %w", err)
	}
	var result []domain.LiveLobbySnapshot
	for _, sess := range sessions {
		phase, err := s.liveSession.GetPhase(ctx, sess.SessionID)
		if err != nil || phase != domain.LivePhaseLobby {
			continue
		}
		result = append(result, sess)
	}
	return result, nil
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

func (s *Service) ListLiveSessions(ctx context.Context, authorID uuid.UUID, source *string, limit int) ([]domain.LiveSession, error) {
	sessions, err := s.sessions.ListLiveSessions(ctx, authorID, source, limit)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	sessionIDs := make([]uuid.UUID, len(sessions))
	for i, sess := range sessions {
		sessionIDs[i] = sess.SessionID
	}
	completedCounts, err := s.attempts.CountCompletedBySessionIDs(ctx, sessionIDs)
	if err != nil {
		return nil, fmt.Errorf("count completed: %w", err)
	}

	for i := range sessions {
		sess := &sessions[i]
		if sess.Status == domain.SessionStatusFinished {
			sessions[i].Phase = domain.LivePhaseCompleted
			sessions[i].ParticipantsCount = completedCounts[sess.SessionID]
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

func (s *Service) GetSessionStatuses(ctx context.Context, ids []uuid.UUID) ([]domain.SessionStatusItem, error) {
	sessions, err := s.sessions.GetManyByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get sessions: %w", err)
	}

	result := make([]domain.SessionStatusItem, 0, len(sessions))
	for i := range sessions {
		sess := &sessions[i]
		item := domain.SessionStatusItem{
			SessionID: sess.ID,
			Mode:      sess.Mode,
			Status:    sess.ComputedStatus(),
		}
		if sess.Mode == domain.SessionModeLive {
			if item.Status == domain.SessionStatusFinished {
				phase := domain.LivePhaseCompleted
				item.Phase = &phase
			} else {
				meta, _ := s.liveSession.GetSessionMeta(ctx, sess.ID)
				if meta == nil || meta.Phase == "" {
					phase := domain.LivePhasePending
					item.Phase = &phase
				} else {
					phase := meta.Phase
					item.Phase = &phase
				}
			}
		}
		result = append(result, item)
	}
	return result, nil
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
