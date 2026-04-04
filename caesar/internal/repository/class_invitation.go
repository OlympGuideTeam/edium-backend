package repository

import (
	"context"
	"database/sql"
	"errors"

	"caesar/internal/domain"

	"github.com/google/uuid"
)

// UpsertInvitation создаёт или возвращает существующее приглашение для (class_id, role).
func (s *PgClassStore) UpsertInvitation(ctx context.Context, classID uuid.UUID, role domain.ClassMemberRole) (uuid.UUID, error) {
	exec := s.db
	var id uuid.UUID
	err := exec.QueryRowContext(ctx, `
		INSERT INTO class_invitation (class_id, role)
		VALUES ($1, $2)
		ON CONFLICT (class_id, role) DO UPDATE SET class_id = EXCLUDED.class_id
		RETURNING id
	`, classID, role).Scan(&id)
	return id, err
}

func (s *PgClassStore) GetInvitation(ctx context.Context, invitationID uuid.UUID) (*domain.ClassInvitation, error) {
	var inv domain.ClassInvitation
	err := s.db.QueryRowContext(ctx,
		`SELECT id, class_id, role FROM class_invitation WHERE id = $1`,
		invitationID,
	).Scan(&inv.ID, &inv.ClassID, &inv.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &inv, err
}
