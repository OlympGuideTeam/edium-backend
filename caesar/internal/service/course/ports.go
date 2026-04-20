package course

import (
	"context"
	"encoding/json"

	"caesar/internal/domain"

	"github.com/google/uuid"
)

type courseStore interface {
	Create(ctx context.Context, classID, ownerID uuid.UUID, title string) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Course, error)
	GetDetail(ctx context.Context, id uuid.UUID) (*domain.CourseDetail, error)
	ListByClassID(ctx context.Context, classID uuid.UUID) ([]domain.CourseListItem, error)
	Update(ctx context.Context, id uuid.UUID, title string) error
	Delete(ctx context.Context, id uuid.UUID) error

	CreateModule(ctx context.Context, courseID uuid.UUID, title string) (uuid.UUID, error)
	GetModuleByID(ctx context.Context, id uuid.UUID) (*domain.CourseModule, error)
	ListModules(ctx context.Context, courseID uuid.UUID) ([]domain.CourseModule, error)
	UpdateModule(ctx context.Context, id uuid.UUID, title string) error
	DeleteModule(ctx context.Context, id uuid.UUID) error
	ReorderModules(ctx context.Context, courseID uuid.UUID, moduleIDs []uuid.UUID) error

	CreateItem(ctx context.Context, moduleID, objectID uuid.UUID, t domain.CourseItemType, payload json.RawMessage) (uuid.UUID, error)
	GetItemByID(ctx context.Context, id uuid.UUID) (*domain.CourseItem, error)
	FindItemByObjectID(ctx context.Context, objectID uuid.UUID) (*domain.CourseItem, error)
	DeleteItem(ctx context.Context, id uuid.UUID) error
	ListItemsByModuleIDs(ctx context.Context, moduleIDs []uuid.UUID, userID uuid.UUID) ([]domain.CourseModuleItem, error)
	UpsertProgress(ctx context.Context, courseItemID, userID, attemptID uuid.UUID) error
	UpdateProgressScore(ctx context.Context, courseItemID, userID uuid.UUID, score float64) error
	UpsertCourseDraft(ctx context.Context, quizTemplateID, courseID uuid.UUID, title string) (uuid.UUID, error)
	GetDraftByID(ctx context.Context, id uuid.UUID) (*domain.CourseDraft, error)
	DeleteDraft(ctx context.Context, id uuid.UUID) error
	ListDraftsByCourseID(ctx context.Context, courseID uuid.UUID) ([]domain.CourseDraft, error)
}

type taskScheduler interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
}

type txRunner interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type classAccessor interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ClassListItem, error)
	GetMemberRole(ctx context.Context, classID, userID uuid.UUID) (domain.ClassMemberRole, bool, error)
}
