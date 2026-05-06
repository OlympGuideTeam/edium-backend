package domain

import (
	"time"

	"github.com/google/uuid"
)

type LiveSource string

const (
	LiveSourceCourse  LiveSource = "course"
	LiveSourceLibrary LiveSource = "library"
)

type LivePhase string

const (
	LivePhasePending        LivePhase = "pending"
	LivePhaseLobby          LivePhase = "lobby"
	LivePhaseQuestionActive LivePhase = "question_active"
	LivePhaseQuestionLocked LivePhase = "question_locked"
	LivePhaseCompleted      LivePhase = "completed"
)

type LiveSessionMeta struct {
	SessionID            uuid.UUID
	QuizTemplateID       uuid.UUID
	AuthorID             uuid.UUID
	QuizTitle            string
	QuestionCount        int
	Source               LiveSource
	Phase                LivePhase
	JoinCode             *string
	QuestionTimeLimitSec int
	IsAnonymousAllowed   bool
	ParticipantsCount    int
}

type LiveParticipant struct {
	AttemptID uuid.UUID
	UserID    *uuid.UUID
	Name      *string
	Status    string
}

type LiveAnswer struct {
	AnswerData  map[string]any
	IsCorrect   bool
	Score       float64
	TimeTakenMs int64
}

type LiveParticipantResult struct {
	Position     int
	AttemptID    uuid.UUID
	UserID       *uuid.UUID
	Name         *string
	Score        float64
	CorrectCount int
	Answers      []LiveAnswerResult
}

type LiveAnswerResult struct {
	QuestionID uuid.UUID
	IsCorrect  bool
	Score      float64
}

type SessionMode string

const (
	SessionModeTest SessionMode = "test"
	SessionModeLive SessionMode = "live"
)

type SessionStatus string

const (
	SessionStatusNotStarted SessionStatus = "not_started"
	SessionStatusWaiting    SessionStatus = "waiting"
	SessionStatusActive     SessionStatus = "active"
	SessionStatusRunning    SessionStatus = "running"
	SessionStatusFinished   SessionStatus = "finished"
)

// ComputedStatus возвращает эффективный статус сессии.
// Для live-сессий статус хранится в БД и возвращается as-is.
// Для test-сессий статус вычисляется из started_at/finished_at:
//
//	now < started_at             → not_started
//	started_at ≤ now ≤ finished_at → active
//	now > finished_at или явный finished → finished
func (s *QuizSession) ComputedStatus() SessionStatus {
	if s.Mode == SessionModeLive {
		return s.Status
	}
	now := time.Now()
	if s.Status == SessionStatusFinished {
		return SessionStatusFinished
	}
	if s.FinishedAt != nil && now.After(*s.FinishedAt) {
		return SessionStatusFinished
	}
	if s.StartedAt != nil && now.Before(*s.StartedAt) {
		return SessionStatusNotStarted
	}
	return SessionStatusActive
}

type QuizSession struct {
	ID                   uuid.UUID
	QuizTemplateID       uuid.UUID
	Mode                 SessionMode
	Source               LiveSource
	LiveHostUserID       *uuid.UUID // ведущий library live; nil — до появления колонки или не library live
	MaxScore             int
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
	Source               LiveSource
	LiveHostUserID       *uuid.UUID
	MaxScore             int
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

type LiveSession struct {
	SessionID         uuid.UUID
	QuizTemplateID    uuid.UUID
	QuizTitle         string
	Source            LiveSource
	Status            SessionStatus
	Phase             LivePhase
	JoinCode          *string
	ParticipantsCount int
	CreatedAt         time.Time
}

type RecentGradeItem struct {
	SessionID      uuid.UUID
	QuizTemplateID uuid.UUID
	QuizTitle      string
	AttemptID      uuid.UUID
	Score          *float64
	Status         AttemptStatus
	FinishedAt     *time.Time
}

type ActiveTestItem struct {
	SessionID         uuid.UUID
	QuizTemplateID    uuid.UUID
	QuizTitle         string
	TotalTimeLimitSec *int
	SessionStartedAt  *time.Time
	SessionFinishedAt *time.Time
	AttemptID         *uuid.UUID
	AttemptStatus     *AttemptStatus
}

type StudentDashboard struct {
	RecentGrades []RecentGradeItem
	ActiveTests  []ActiveTestItem
}

type LiveLobbySnapshot struct {
	SessionID            uuid.UUID
	CourseID             uuid.UUID
	QuizTitle            string
	QuestionTimeLimitSec int
}

type AwaitingReviewSession struct {
	SessionID      uuid.UUID
	QuizTemplateID uuid.UUID
	QuizTitle      string
	GradingCount   int
	GradedCount    int
	CompletedCount int
}

type SessionStatusItem struct {
	SessionID uuid.UUID
	Mode      SessionMode
	Status    SessionStatus
	Phase     *LivePhase // nil для test-сессий
}

type CreateTestCourseSessionInlineParams struct {
	Title             string
	Description       *string
	CourseID          uuid.UUID
	ModuleID          uuid.UUID
	Questions         []AddQuestionParams
	TotalTimeLimitSec *int
	ShuffleQuestions  *bool
	StartedAt         *time.Time
	FinishedAt        *time.Time
}

type CreateLiveCourseSessionInlineParams struct {
	Title                string
	Description          *string
	CourseID             uuid.UUID
	ModuleID             uuid.UUID
	Questions            []AddQuestionParams
	QuestionTimeLimitSec *int
}
