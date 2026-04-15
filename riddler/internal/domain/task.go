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

const (
	TaskTypeQuizTemplateAttachedPublisher TaskType = "quiz_template_attached_publisher"
	TaskTypeCourseSessionCreatedPublisher TaskType = "course_session_created_publisher"
	TaskTypeAttemptCreatedPublisher       TaskType = "attempt_created_publisher"
	TaskTypeAttemptScoredPublisher        TaskType = "attempt_scored_publisher"
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
