package domain

import (
	"time"

	"github.com/google/uuid"
)

type NotificationData struct {
	Route *string `json:"route,omitempty"`
	Role  *string `json:"role,omitempty"`
}

type Notification struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Title     string
	Body      string
	IsRead    bool
	Data      *NotificationData
	CreatedAt time.Time
}
