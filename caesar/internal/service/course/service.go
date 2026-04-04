package course

import (
	"context"
	"fmt"
	"strings"

	"caesar/internal/domain"

	"github.com/google/uuid"
)

type Service struct {
	courses courseStore
	classes classAccessor
}

func NewService(courses courseStore, classes classAccessor) *Service {
	return &Service{courses: courses, classes: classes}
}

func (s *Service) CreateCourse(ctx context.Context, classID, userID uuid.UUID, title string) (uuid.UUID, error) {
	if strings.TrimSpace(title) == "" {
		return uuid.Nil, ErrEmptyTitle
	}

	role, ok, err := s.classes.GetMemberRole(ctx, classID, userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("getMemberRole: %w", err)
	}
	if !ok {
		return uuid.Nil, ErrNotMember
	}
	if role == domain.ClassMemberRoleStudent {
		return uuid.Nil, ErrForbidden
	}

	id, err := s.courses.Create(ctx, classID, userID, title)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create: %w", err)
	}
	return id, nil
}

func (s *Service) GetCourse(ctx context.Context, courseID, userID uuid.UUID) (*domain.CourseDetail, error) {
	detail, err := s.courses.GetDetail(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("getDetail: %w", err)
	}
	if detail == nil {
		return nil, ErrNotFound
	}

	role, ok, err := s.classes.GetMemberRole(ctx, detail.ClassID, userID)
	if err != nil {
		return nil, fmt.Errorf("getMemberRole: %w", err)
	}
	if !ok {
		return nil, ErrNotMember
	}

	modules, err := s.courses.ListModules(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("listModules: %w", err)
	}
	detail.Modules = modules
	detail.IsTeacher = role != domain.ClassMemberRoleStudent

	return detail, nil
}

func (s *Service) GetClassCourses(ctx context.Context, classID, userID uuid.UUID) ([]domain.CourseListItem, error) {
	cl, err := s.classes.GetByID(ctx, classID)
	if err != nil {
		return nil, fmt.Errorf("getClass: %w", err)
	}
	if cl == nil {
		return nil, ErrNotFound
	}

	role, ok, err := s.classes.GetMemberRole(ctx, classID, userID)
	if err != nil {
		return nil, fmt.Errorf("getMemberRole: %w", err)
	}
	if !ok {
		return nil, ErrNotMember
	}

	isTeacher := role != domain.ClassMemberRoleStudent

	items, err := s.courses.ListByClassID(ctx, classID)
	if err != nil {
		return nil, fmt.Errorf("listByClassID: %w", err)
	}
	for i := range items {
		items[i].IsTeacher = isTeacher
	}
	return items, nil
}

func (s *Service) UpdateCourse(ctx context.Context, courseID, userID uuid.UUID, title string) error {
	if strings.TrimSpace(title) == "" {
		return ErrEmptyTitle
	}

	c, err := s.getCourse(ctx, courseID)
	if err != nil {
		return err
	}
	if err := s.requireCanModify(ctx, c, userID); err != nil {
		return err
	}

	if err := s.courses.Update(ctx, courseID, title); err != nil {
		return fmt.Errorf("update: %w", err)
	}
	return nil
}

func (s *Service) DeleteCourse(ctx context.Context, courseID, userID uuid.UUID) error {
	c, err := s.getCourse(ctx, courseID)
	if err != nil {
		return err
	}
	if err := s.requireCanModify(ctx, c, userID); err != nil {
		return err
	}

	if err := s.courses.Delete(ctx, courseID); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

func (s *Service) getCourse(ctx context.Context, id uuid.UUID) (*domain.Course, error) {
	c, err := s.courses.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getByID: %w", err)
	}
	if c == nil {
		return nil, ErrNotFound
	}
	return c, nil
}

// requireCanModify проверяет, что пользователь — создатель курса или владелец класса.
func (s *Service) requireCanModify(ctx context.Context, c *domain.Course, userID uuid.UUID) error {
	if c.OwnerID == userID {
		return nil
	}
	cl, err := s.classes.GetByID(ctx, c.ClassID)
	if err != nil {
		return fmt.Errorf("getClass: %w", err)
	}
	if cl == nil || cl.OwnerID != userID {
		return ErrForbidden
	}
	return nil
}
