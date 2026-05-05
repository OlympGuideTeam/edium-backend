package live

import (
	"context"
	"time"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/repository"
	livesvc "riddler/internal/service/live"
)

type liveService interface {
	CreateLiveCourseSession(ctx context.Context, authorID, quizTemplateID, moduleID uuid.UUID, questionTimeLimitSec *int) (uuid.UUID, error)
	CreateLiveLibrarySession(ctx context.Context, authorID, quizTemplateID uuid.UUID, questionTimeLimitSec *int) (uuid.UUID, error)
	ListLibraryLiveSessions(ctx context.Context, authorID uuid.UUID) ([]domain.LibraryLiveSession, error)
StartLiveSession(ctx context.Context, sessionID, authorID uuid.UUID) (wsToken, joinCode string, err error)
	ResolveLiveCode(ctx context.Context, code string) (*domain.LiveSessionMeta, error)
	JoinLiveSession(ctx context.Context, sessionID uuid.UUID, userID *uuid.UUID, name *string) (attemptID uuid.UUID, wsToken string, err error)
	GetSessionStatuses(ctx context.Context, ids []uuid.UUID) ([]domain.SessionStatusItem, error)
	GetLiveResultsStudent(ctx context.Context, sessionID, attemptID uuid.UUID) (*livesvc.StudentResults, error)
	GetLiveResultsTeacher(ctx context.Context, sessionID, callerID uuid.UUID) (*livesvc.TeacherResults, error)

	// WS
	ConsumeWsToken(ctx context.Context, sessionID uuid.UUID, token string) (*repository.WsTokenPayload, error)
	IsKicked(ctx context.Context, sessionID, attemptID uuid.UUID) (bool, error)
	GetSessionMeta(ctx context.Context, sessionID uuid.UUID) (*domain.LiveSessionMeta, error)
	GetQuestions(ctx context.Context, sessionID uuid.UUID) ([]domain.QuestionWithOptions, error)
	GetParticipants(ctx context.Context, sessionID uuid.UUID) ([]domain.LiveParticipant, error)
	SetParticipantStatus(ctx context.Context, sessionID, attemptID uuid.UUID, status string) error
	SetKicked(ctx context.Context, attemptID uuid.UUID) error
	GetPhase(ctx context.Context, sessionID uuid.UUID) (domain.LivePhase, error)
	SetPhase(ctx context.Context, sessionID uuid.UUID, phase domain.LivePhase) error
	SetCurrentQuestion(ctx context.Context, sessionID uuid.UUID, idx int, deadline time.Time) error
	GetCurrentQuestion(ctx context.Context, sessionID uuid.UUID) (int, time.Time, error)
	SaveAnswer(ctx context.Context, sessionID, questionID, attemptID uuid.UUID, ans domain.LiveAnswer) error
	GetAnswer(ctx context.Context, sessionID, questionID, attemptID uuid.UUID) (*domain.LiveAnswer, error)
	GetAllAnswers(ctx context.Context, sessionID, questionID uuid.UUID) (map[uuid.UUID]domain.LiveAnswer, error)
	GetAnsweredCount(ctx context.Context, sessionID, questionID uuid.UUID) (int, error)
	CompleteLiveSession(ctx context.Context, sessionID uuid.UUID) error
}
