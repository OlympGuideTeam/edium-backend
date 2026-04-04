package repository

import (
	"context"

	"caesar/internal/domain"
	"caesar/internal/infra/db"
	"caesar/internal/pkg/apperr"

	"github.com/google/uuid"
)

func (s *PgClassStore) AddMember(ctx context.Context, classID, userID uuid.UUID, role domain.ClassMemberRole) error {
	exec := db.ExecutorFromContext(ctx, s.db)
	_, err := exec.ExecContext(ctx,
		`INSERT INTO class_member (class_id, user_id, role) VALUES ($1, $2, $3)
		 ON CONFLICT (class_id, user_id) DO NOTHING`,
		classID, userID, role,
	)
	return err
}

func (s *PgClassStore) GetMembersForDetail(ctx context.Context, classID uuid.UUID) ([]domain.ClassMember, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT cm.class_id, cm.user_id, u.name, u.surname, cm.role
		FROM class_member cm
		JOIN "user" u ON u.id = cm.user_id
		WHERE cm.class_id = $1
		ORDER BY u.surname, u.name
	`, classID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var members []domain.ClassMember
	for rows.Next() {
		var m domain.ClassMember
		if err := rows.Scan(&m.ClassID, &m.UserID, &m.Name, &m.Surname, &m.Role); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (s *PgClassStore) RemoveMember(ctx context.Context, classID, userID uuid.UUID) error {
	exec := db.ExecutorFromContext(ctx, s.db)
	res, err := exec.ExecContext(ctx,
		`DELETE FROM class_member WHERE class_id = $1 AND user_id = $2`,
		classID, userID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return apperr.ErrMemberNotFound
	}
	return nil
}

func (s *PgClassStore) IsMember(ctx context.Context, classID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM class_member WHERE class_id = $1 AND user_id = $2)`,
		classID, userID,
	).Scan(&exists)
	return exists, err
}
