package class

import (
	"context"

	"caesar/internal/pkg/apperr"

	"github.com/google/uuid"
)

func (s *Service) RemoveMember(ctx context.Context, classID, userID, targetUserID uuid.UUID) error {
	c, err := s.getClass(ctx, classID)
	if err != nil {
		return err
	}
	if err := s.requireOwner(c, userID); err != nil {
		return err
	}
	if targetUserID == c.OwnerID {
		return apperr.ErrClassRemoveOwner
	}
	return s.classes.RemoveMember(ctx, classID, targetUserID)
}
