package course

import (
	"context"
	"fmt"

	"caesar/internal/pkg/apperr"

	"github.com/google/uuid"
)

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
