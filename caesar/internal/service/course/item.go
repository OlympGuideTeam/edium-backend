package course

import (
	"context"
	"encoding/json"
	"fmt"

	"caesar/internal/domain"
	"caesar/internal/pkg/apperr"

	"github.com/google/uuid"
)

type courseSessionDeletedPayload struct {
	SessionID uuid.UUID `json:"session_id"`
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

	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.courses.DeleteItem(ctx, itemID); err != nil {
			return fmt.Errorf("deleteItem: %w", err)
		}
		if item.Type == domain.CourseItemTypeQuiz {
			payload, _ := json.Marshal(courseSessionDeletedPayload{SessionID: item.ObjectID})
			if err := s.tasks.Schedule(ctx, domain.CourseSessionDeleted, payload); err != nil {
				return fmt.Errorf("schedule course_session_deleted: %w", err)
			}
		}
		return nil
	})
}

func (s *Service) DeleteCourseDraft(ctx context.Context, draftID, userID uuid.UUID) error {
	draft, err := s.courses.GetDraftByID(ctx, draftID)
	if err != nil {
		return fmt.Errorf("getDraftByID: %w", err)
	}
	if draft == nil {
		return apperr.ErrCourseItemNotFound
	}

	c, err := s.getCourse(ctx, draft.CourseID)
	if err != nil {
		return err
	}
	if err := s.requireCanModify(ctx, c, userID); err != nil {
		return err
	}

	if err := s.courses.DeleteDraft(ctx, draftID); err != nil {
		return fmt.Errorf("deleteDraft: %w", err)
	}
	return nil
}
