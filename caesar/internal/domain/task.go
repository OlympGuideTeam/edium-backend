package domain

import (
	"time"

	"github.com/google/uuid"
)

type TaskStatus string

var (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusProcessing TaskStatus = "processing"
	TaskStatusDone       TaskStatus = "done"
	TaskStatusFailed     TaskStatus = "failed"
)

type TaskType string

var (
	UserCreated TaskType = "user_created"
	UserDeleted TaskType = "user_deleted"

	QuizTemplateAttached TaskType = "quiz_template_attached"
	CourseSessionCreated TaskType = "course_session_created"
	AttemptCreated       TaskType = "attempt_created"
	AttemptScored        TaskType = "attempt_scored"
)

type Task struct {
	ID       uuid.UUID
	Type     TaskType
	Status   TaskStatus
	Payload  []byte
	TraceCtx string

	Attempts    int64
	AvailableAt time.Time
	MaxAttempts *int64
	LastError   *error
}
