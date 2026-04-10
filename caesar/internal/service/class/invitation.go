package class

import (
	"context"
	"fmt"

	"caesar/internal/domain"
	"caesar/internal/pkg/apperr"

	"github.com/google/uuid"
)

func (s *Service) GetInviteLink(ctx context.Context, classID, userID uuid.UUID, role domain.ClassMemberRole) (uuid.UUID, error) {
	c, err := s.getClass(ctx, classID)
	if err != nil {
		return uuid.Nil, err
	}
	if err := s.requireOwner(c, userID); err != nil {
		return uuid.Nil, err
	}
	invitationID, err := s.classes.UpsertInvitation(ctx, classID, role)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsertInvitation: %w", err)
	}
	return invitationID, nil
}

func (s *Service) AcceptInvitation(ctx context.Context, invitationID, userID uuid.UUID) error {
	inv, err := s.classes.GetInvitation(ctx, invitationID)
	if err != nil {
		return fmt.Errorf("getInvitation: %w", err)
	}
	if inv == nil {
		return apperr.ErrInvitationNotFound
	}

	c, err := s.getClass(ctx, inv.ClassID)
	if err != nil {
		return err
	}
	if c.OwnerID == userID {
		return apperr.ErrAlreadyMember
	}

	isMember, err := s.classes.IsMember(ctx, inv.ClassID, userID)
	if err != nil {
		return fmt.Errorf("isMember: %w", err)
	}
	if isMember {
		return apperr.ErrAlreadyMember
	}

	if err := s.classes.AddMember(ctx, inv.ClassID, userID, inv.Role); err != nil {
		return fmt.Errorf("addMember: %w", err)
	}
	return nil
}
