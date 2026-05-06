package domain

import (
	"time"

	"github.com/google/uuid"
)

type FCMDevice struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	FCMToken  string
	Platform  string
	CreatedAt time.Time
}
