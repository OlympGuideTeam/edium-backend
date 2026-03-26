package domain

import "time"

type PendingOTP struct {
	Phone     string
	ChatID    int64
	ExpiresAt time.Time
}
