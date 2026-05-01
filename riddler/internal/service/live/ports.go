package live

import (
	"context"

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
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.SessionStatus) error
}

type attemptRepository interface {
	CreateLiveAttempt(ctx context.Context, sessionID uuid.UUID, userID *uuid.UUID, name *string) (uuid.UUID, error)
	GetBySessionAndUser(ctx context.Context, sessionID, userID uuid.UUID) (*domain.Attempt, error)
	GetLiveLeaderboard(ctx context.Context, sessionID uuid.UUID) ([]domain.LiveParticipantResult, error)
	GetLiveSessionAnswers(ctx context.Context, sessionID uuid.UUID) ([]repository.LiveSessionAnswer, error)
}

type liveRepository interface {
	InitSession(ctx context.Context, sessionID uuid.UUID, quiz *domain.QuizTemplate, timeLimitSec int, source domain.LiveSource, authorID uuid.UUID) (joinCode string, err error)
	GetPhase(ctx context.Context, sessionID uuid.UUID) (domain.LivePhase, error)
	IsKicked(ctx context.Context, sessionID, attemptID uuid.UUID) (bool, error)
	AddParticipant(ctx context.Context, sessionID uuid.UUID, p domain.LiveParticipant) error
	IssueWsToken(ctx context.Context, sessionID uuid.UUID, payload repository.WsTokenPayload) (string, error)
	ResolveCode(ctx context.Context, code string) (uuid.UUID, error)
	GetSessionMeta(ctx context.Context, sessionID uuid.UUID) (*domain.LiveSessionMeta, error)
}

type taskScheduler interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
}

type txManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
