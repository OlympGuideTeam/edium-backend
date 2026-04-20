package course

import (
	"context"
	"fmt"
	"strings"

	"caesar/internal/domain"
	"caesar/internal/pkg/apperr"

	"github.com/google/uuid"
)

func (s *Service) GetModule(ctx context.Context, moduleID, userID uuid.UUID) (*domain.CourseModule, error) {
	m, err := s.courses.GetModuleByID(ctx, moduleID)
	if err != nil {
		return nil, fmt.Errorf("getModuleByID: %w", err)
	}
	if m == nil {
		return nil, apperr.ErrModuleNotFound
	}

	c, err := s.getCourse(ctx, m.CourseID)
	if err != nil {
		return nil, err
	}

	_, ok, err := s.classes.GetMemberRole(ctx, c.ClassID, userID)
	if err != nil {
		return nil, fmt.Errorf("getMemberRole: %w", err)
	}
	if !ok {
		return nil, apperr.ErrCourseNotMember
	}

	items, err := s.courses.ListItemsByModuleIDs(ctx, []uuid.UUID{moduleID}, userID)
	if err != nil {
		return nil, fmt.Errorf("listItems: %w", err)
	}
	m.Items = items
	return m, nil
}

func (s *Service) CreateModule(ctx context.Context, courseID, userID uuid.UUID, title string) (uuid.UUID, error) {
	if strings.TrimSpace(title) == "" {
		return uuid.Nil, apperr.ErrCourseEmptyTitle
	}

	c, err := s.getCourse(ctx, courseID)
	if err != nil {
		return uuid.Nil, err
	}
	if err := s.requireCanModify(ctx, c, userID); err != nil {
		return uuid.Nil, err
	}

	id, err := s.courses.CreateModule(ctx, courseID, title)
	if err != nil {
		return uuid.Nil, fmt.Errorf("createModule: %w", err)
	}
	return id, nil
}

func (s *Service) UpdateModule(ctx context.Context, moduleID, userID uuid.UUID, title string) error {
	if strings.TrimSpace(title) == "" {
		return apperr.ErrCourseEmptyTitle
	}

	m, err := s.courses.GetModuleByID(ctx, moduleID)
	if err != nil {
		return fmt.Errorf("getModuleByID: %w", err)
	}
	if m == nil {
		return apperr.ErrModuleNotFound
	}

	c, err := s.getCourse(ctx, m.CourseID)
	if err != nil {
		return err
	}
	if err := s.requireCanModify(ctx, c, userID); err != nil {
		return err
	}

	if err := s.courses.UpdateModule(ctx, moduleID, title); err != nil {
		return fmt.Errorf("updateModule: %w", err)
	}
	return nil
}

func (s *Service) ReorderModules(ctx context.Context, courseID, userID uuid.UUID, moduleIDs []uuid.UUID) error {
	c, err := s.getCourse(ctx, courseID)
	if err != nil {
		return err
	}
	if err := s.requireCanModify(ctx, c, userID); err != nil {
		return err
	}
	if err := s.courses.ReorderModules(ctx, courseID, moduleIDs); err != nil {
		return fmt.Errorf("reorderModules: %w", err)
	}
	return nil
}

func (s *Service) DeleteModule(ctx context.Context, moduleID, userID uuid.UUID) error {
	m, err := s.courses.GetModuleByID(ctx, moduleID)
	if err != nil {
		return fmt.Errorf("getModuleByID: %w", err)
	}
	if m == nil {
		return apperr.ErrModuleNotFound
	}

	c, err := s.getCourse(ctx, m.CourseID)
	if err != nil {
		return err
	}
	if err := s.requireCanModify(ctx, c, userID); err != nil {
		return err
	}

	if err := s.courses.DeleteModule(ctx, moduleID); err != nil {
		return fmt.Errorf("deleteModule: %w", err)
	}
	return nil
}
