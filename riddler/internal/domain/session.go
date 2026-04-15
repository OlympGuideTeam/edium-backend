package domain

import (
	"time"

	"github.com/google/uuid"
)

type SessionMode string

const (
	SessionModeTest SessionMode = "test"
	SessionModeLive SessionMode = "live"
)

type SessionStatus string

const (
	SessionStatusDraft    SessionStatus = "draft"
	SessionStatusWaiting  SessionStatus = "waiting"
	SessionStatusActive   SessionStatus = "active"
	SessionStatusFinished SessionStatus = "finished"
)

type QuizSession struct {
	ID                   uuid.UUID
	QuizTemplateID       uuid.UUID
	Mode                 SessionMode
	TotalTimeLimitSec    *int
	QuestionTimeLimitSec *int
	ShuffleQuestions     *bool
	Status               SessionStatus
	Settings             map[string]any
	StartedAt            *time.Time
	FinishedAt           *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type CreateSessionParams struct {
	QuizTemplateID       uuid.UUID
	Mode                 SessionMode
	TotalTimeLimitSec    *int
	QuestionTimeLimitSec *int
	ShuffleQuestions     *bool
	Status               SessionStatus
	Settings             map[string]any
	StartedAt            *time.Time
	FinishedAt           *time.Time
}

type CreateTestCourseSessionParams struct {
	TotalTimeLimitSec *int
	ShuffleQuestions  *bool
	StartedAt         *time.Time
	FinishedAt        *time.Time
}

type CreateLiveCourseSessionParams struct {
	QuestionTimeLimitSec *int
}
