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
	CompletionRequested TaskType = "completion_requested"
	CompletionCompleted TaskType = "completion_completed"

	QuizGradeRequested TaskType = "quiz_grade_requested"
	QuizGradeCompleted TaskType = "quiz_grade_completed"
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
