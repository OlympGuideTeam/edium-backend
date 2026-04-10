package domain

import (
	"time"

	"github.com/google/uuid"
)

type SMSTaskStatus string

const (
	SMSTaskStatusPending SMSTaskStatus = "pending"
	SMSTaskStatusSent    SMSTaskStatus = "sent"
	SMSTaskStatusFailed  SMSTaskStatus = "failed"
)

type SMSTask struct {
	ID          uuid.UUID
	Phone       string
	Text        string
	Status      SMSTaskStatus
	CreatedAt   time.Time
	ProcessedAt *time.Time
	ClaimedAt   *time.Time
	RetryCount  int
	MaxRetries  int
}
