package course

import (
	"context"

	"caesar/internal/domain"

	"github.com/google/uuid"
)

type courseService interface {
	CreateCourse(ctx context.Context, classID, userID uuid.UUID, title string) (uuid.UUID, error)
	GetCourse(ctx context.Context, courseID, userID uuid.UUID) (*domain.CourseDetail, error)
	GetClassCourses(ctx context.Context, classID, userID uuid.UUID) ([]domain.CourseListItem, error)
	GetMyCourses(ctx context.Context, userID uuid.UUID) ([]domain.CourseListItem, error)
	UpdateCourse(ctx context.Context, courseID, userID uuid.UUID, title string) error
	DeleteCourse(ctx context.Context, courseID, userID uuid.UUID) error

	GetModule(ctx context.Context, moduleID, userID uuid.UUID) (*domain.CourseModule, error)
	GetModuleRoster(ctx context.Context, moduleID, userID uuid.UUID) ([]domain.ClassMember, error)
	CreateModule(ctx context.Context, courseID, userID uuid.UUID, title string) (uuid.UUID, error)
	UpdateModule(ctx context.Context, moduleID, userID uuid.UUID, title string) error
	DeleteModule(ctx context.Context, moduleID, userID uuid.UUID) error
	ReorderModules(ctx context.Context, courseID, userID uuid.UUID, moduleIDs []uuid.UUID) error

	DeleteCourseItem(ctx context.Context, itemID, userID uuid.UUID) error
	DeleteCourseDraft(ctx context.Context, draftID, userID uuid.UUID) error

	GetCourseSheet(ctx context.Context, courseID, userID uuid.UUID) (*domain.CourseSheet, error)
}
