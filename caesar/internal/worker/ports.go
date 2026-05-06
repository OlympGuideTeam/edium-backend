package worker

import (
	"caesar/internal/domain"
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type taskScheduler interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
}

type taskRepository interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
	ClaimPending(ctx context.Context, taskType domain.TaskType, limit int) ([]domain.Task, error)
	MarkDone(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string, retryAfter time.Duration) error
}

type courseItemStore interface {
	CreateItem(ctx context.Context, moduleID, objectID uuid.UUID, t domain.CourseItemType, payload json.RawMessage) (uuid.UUID, error)
	FindItemByObjectID(ctx context.Context, objectID uuid.UUID) (*domain.CourseItem, error)
	DeleteItem(ctx context.Context, id uuid.UUID) error
	UpsertProgress(ctx context.Context, courseItemID, userID, attemptID uuid.UUID) error
	UpdateProgressScore(ctx context.Context, courseItemID, userID uuid.UUID, score float64) error
}

type courseDraftStore interface {
	UpsertCourseDraft(ctx context.Context, quizTemplateID, courseID uuid.UUID, title string, payload json.RawMessage) (uuid.UUID, error)
	DeleteDraftByTemplateAndCourse(ctx context.Context, quizTemplateID, courseID uuid.UUID) error
}

type courseStudentStore interface {
	GetStudentIDsByCourseID(ctx context.Context, courseID uuid.UUID) ([]uuid.UUID, error)
}
