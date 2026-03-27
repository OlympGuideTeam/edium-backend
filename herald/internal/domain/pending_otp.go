package domain

import "time"

type PendingOTP struct {
	Phone     string
	Channel   Channel
	ChatID    int64
	ExpiresAt time.Time
}
