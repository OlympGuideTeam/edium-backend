package live

import (
	"context"
	"time"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/repository"
)

type quizRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.QuizTemplate, error)
	HasFreeAnswerQuestions(ctx context.Context, quizID uuid.UUID) (bool, error)
	GetQuestionsWithOptions(ctx context.Context, quizID uuid.UUID) ([]domain.QuestionWithOptions, error)
}

type sessionRepository interface {
	Create(ctx context.Context, p domain.CreateSessionParams) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.QuizSession, error)
	GetManyByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.QuizSession, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.SessionStatus) error
	ListLiveSessions(ctx context.Context, authorID uuid.UUID, source *string, limit int) ([]domain.LiveSession, error)
	FindRunningCourseLiveSessions(ctx context.Context, courseIDs []uuid.UUID) ([]domain.LiveLobbySnapshot, error)
}

type attemptRepository interface {
	CreateLiveAttempt(ctx context.Context, sessionID uuid.UUID, userID *uuid.UUID, name *string) (uuid.UUID, error)
	GetBySessionAndUser(ctx context.Context, sessionID, userID uuid.UUID) (*domain.Attempt, error)
	GetLiveLeaderboard(ctx context.Context, sessionID uuid.UUID) ([]domain.LiveParticipantResult, error)
	GetLiveSessionAnswers(ctx context.Context, sessionID uuid.UUID) ([]repository.LiveSessionAnswer, error)
	BulkInsertSubmissions(ctx context.Context, submissions []repository.BulkSubmission) error
	PublishLive(ctx context.Context, attemptID uuid.UUID, score, grade float64) error
	SetKicked(ctx context.Context, attemptID uuid.UUID) error
	CountCompletedBySessionIDs(ctx context.Context, sessionIDs []uuid.UUID) (map[uuid.UUID]int, error)
}

type liveSessionStore interface {
	InitSession(ctx context.Context, sessionID uuid.UUID, quiz *domain.QuizTemplate, timeLimitSec int, source domain.LiveSource, authorID uuid.UUID) (joinCode string, err error)
	GetSessionMeta(ctx context.Context, sessionID uuid.UUID) (*domain.LiveSessionMeta, error)
	GetPhase(ctx context.Context, sessionID uuid.UUID) (domain.LivePhase, error)
	SetPhase(ctx context.Context, sessionID uuid.UUID, phase domain.LivePhase) error
	ResolveCode(ctx context.Context, code string) (uuid.UUID, error)
	SetCurrentQuestion(ctx context.Context, sessionID uuid.UUID, idx int, deadline time.Time) error
	GetCurrentQuestion(ctx context.Context, sessionID uuid.UUID) (int, time.Time, error)
	DeleteAll(ctx context.Context, sessionID uuid.UUID, code string) error
}

type liveParticipantStore interface {
	AddParticipant(ctx context.Context, sessionID uuid.UUID, p domain.LiveParticipant) error
	GetParticipants(ctx context.Context, sessionID uuid.UUID) ([]domain.LiveParticipant, error)
	SetParticipantStatus(ctx context.Context, sessionID, attemptID uuid.UUID, status string) error
	IsKicked(ctx context.Context, sessionID, attemptID uuid.UUID) (bool, error)
}

type liveAnswerStore interface {
	SaveAnswer(ctx context.Context, sessionID, questionID, attemptID uuid.UUID, ans domain.LiveAnswer) error
	GetAnswer(ctx context.Context, sessionID, questionID, attemptID uuid.UUID) (*domain.LiveAnswer, error)
	GetAllAnswers(ctx context.Context, sessionID, questionID uuid.UUID) (map[uuid.UUID]domain.LiveAnswer, error)
	GetAnsweredCount(ctx context.Context, sessionID, questionID uuid.UUID) (int, error)
}

type liveTokenStore interface {
	IssueWsToken(ctx context.Context, sessionID uuid.UUID, payload repository.WsTokenPayload) (string, error)
	ConsumeWsToken(ctx context.Context, sessionID uuid.UUID, token string) (*repository.WsTokenPayload, error)
}

type taskScheduler interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
}

type studentNotifier interface {
	NotifyLobbyOpened(sessionID, courseID uuid.UUID, quizTitle string, questionTimeLimitSec int)
}

type txManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
