package class

import (
	"context"

	"caesar/internal/domain"

	"github.com/google/uuid"
)

type classStore interface {
	Create(ctx context.Context, ownerID uuid.UUID, title string) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ClassListItem, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, role domain.ClassMemberRole) ([]domain.ClassListItem, error)
	Update(ctx context.Context, id uuid.UUID, title string) error
	Delete(ctx context.Context, id uuid.UUID) error

	AddMember(ctx context.Context, classID, userID uuid.UUID, role domain.ClassMemberRole) error
	GetMembersForDetail(ctx context.Context, classID uuid.UUID) ([]domain.ClassMember, error)
	RemoveMember(ctx context.Context, classID, userID uuid.UUID) error
	IsMember(ctx context.Context, classID, userID uuid.UUID) (bool, error)

	UpsertInvitation(ctx context.Context, classID uuid.UUID, role domain.ClassMemberRole) (uuid.UUID, error)
	GetInvitation(ctx context.Context, invitationID uuid.UUID) (*domain.ClassInvitation, error)
}

type courseAccessor interface {
	ListByClassID(ctx context.Context, classID uuid.UUID) ([]domain.CourseListItem, error)
}
