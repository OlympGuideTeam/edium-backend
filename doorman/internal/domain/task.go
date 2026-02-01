package domain

import (
	"github.com/google/uuid"
	"time"
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
	OTPSent     TaskType = "otp_sent"
)

type Task struct {
	ID      uuid.UUID
	Type    TaskType
	Status  TaskStatus
	Payload []byte

	Attempts    int64
	AvailableAt time.Time
	MaxAttempts *int64
	LastError   *error
}
