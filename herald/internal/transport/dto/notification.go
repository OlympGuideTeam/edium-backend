package dto

import "time"

type NotificationDataResponse struct {
	Route *string `json:"route,omitempty"`
	Role  *string `json:"role,omitempty"`
}

type NotificationResponse struct {
	ID        string                    `json:"id"`
	Title     string                    `json:"title"`
	Body      string                    `json:"body"`
	CreatedAt time.Time                 `json:"created_at"`
	IsRead    bool                      `json:"is_read"`
	Data      *NotificationDataResponse `json:"data"`
}
