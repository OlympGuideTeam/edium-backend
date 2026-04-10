package course

import (
	"context"
	"fmt"

	"caesar/internal/domain"
	"caesar/internal/pkg/apperr"

	"github.com/google/uuid"
)

func (s *Service) CreateCourseItem(ctx context.Context, moduleID, userID, refID uuid.UUID, t domain.CourseItemType, orderIndex int) (uuid.UUID, error) {
	m, err := s.courses.GetModuleByID(ctx, moduleID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("getModuleByID: %w", err)
	}
	if m == nil {
		return uuid.Nil, apperr.ErrModuleNotFound
	}

	c, err := s.getCourse(ctx, m.CourseID)
	if err != nil {
		return uuid.Nil, err
	}
	if err := s.requireCanModify(ctx, c, userID); err != nil {
		return uuid.Nil, err
	}

	id, err := s.courses.CreateItem(ctx, moduleID, refID, t, orderIndex)
	if err != nil {
		return uuid.Nil, fmt.Errorf("createItem: %w", err)
	}
	return id, nil
}

func (s *Service) DeleteCourseItem(ctx context.Context, itemID, userID uuid.UUID) error {
	item, err := s.courses.GetItemByID(ctx, itemID)
	if err != nil {
		return fmt.Errorf("getItemByID: %w", err)
	}
	if item == nil {
		return apperr.ErrCourseItemNotFound
	}

	m, err := s.courses.GetModuleByID(ctx, item.ModuleID)
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

	if err := s.courses.DeleteItem(ctx, itemID); err != nil {
		return fmt.Errorf("deleteItem: %w", err)
	}
	return nil
}
