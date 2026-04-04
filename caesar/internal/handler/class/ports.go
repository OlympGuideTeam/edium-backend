package class

import (
	"context"

	"caesar/internal/domain"

	"github.com/google/uuid"
)

type classService interface {
	GetMyClasses(ctx context.Context, userID uuid.UUID, role domain.ClassMemberRole) ([]domain.ClassListItem, error)
	CreateClass(ctx context.Context, ownerID uuid.UUID, title string) (uuid.UUID, error)
	GetClass(ctx context.Context, classID, userID uuid.UUID) (*domain.ClassDetail, error)
	UpdateClass(ctx context.Context, classID, userID uuid.UUID, title string) error
	DeleteClass(ctx context.Context, classID, userID uuid.UUID) error
	RemoveMember(ctx context.Context, classID, userID, targetUserID uuid.UUID) error
	GetInviteLink(ctx context.Context, classID, userID uuid.UUID, role domain.ClassMemberRole) (uuid.UUID, error)
	AcceptInvitation(ctx context.Context, invitationID, userID uuid.UUID) error
}
