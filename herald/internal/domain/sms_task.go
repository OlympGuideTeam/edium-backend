package domain

import (
	"time"

	"github.com/google/uuid"
)

type SMSTask struct {
	ID          uuid.UUID
	Phone       string
	Text        string
	CreatedAt   time.Time
	ProcessedAt *time.Time
	RetryCount  int
	MaxRetries  int
}
