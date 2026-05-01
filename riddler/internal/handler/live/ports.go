package live

import (
	"context"

	"github.com/google/uuid"

	"riddler/internal/domain"
	livesvc "riddler/internal/service/live"
)

type liveService interface {
	CreateLiveCourseSession(ctx context.Context, authorID, quizTemplateID, moduleID uuid.UUID, questionTimeLimitSec *int) (uuid.UUID, error)
	CreateLiveLibrarySession(ctx context.Context, authorID, quizTemplateID uuid.UUID, questionTimeLimitSec *int) (uuid.UUID, error)
	StartLiveSession(ctx context.Context, sessionID, authorID uuid.UUID) (wsToken, joinCode string, err error)
	ResolveLiveCode(ctx context.Context, code string) (*domain.LiveSessionMeta, error)
	JoinLiveSession(ctx context.Context, sessionID uuid.UUID, userID *uuid.UUID, name *string) (attemptID uuid.UUID, wsToken string, err error)
	GetLiveResultsStudent(ctx context.Context, sessionID, attemptID uuid.UUID) (*livesvc.StudentResults, error)
	GetLiveResultsTeacher(ctx context.Context, sessionID, callerID uuid.UUID) (*livesvc.TeacherResults, error)
}
