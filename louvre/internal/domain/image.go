package domain

import (
	"time"

	"github.com/google/uuid"
)

type Image struct {
	ID         uuid.UUID
	FileName   string
	MimeType   string
	SizeBytes  int64
	Width      *int
	Height     *int
	S3Key      string
	BucketName string
	CreatedAt  time.Time
	DeletedAt  *time.Time
}
