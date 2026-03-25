package domain

import "time"

type PendingOTP struct {
	CorrelationID string
	ChatID        int64
	ExpiresAt     time.Time
}
