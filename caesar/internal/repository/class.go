package repository

import (
	"context"
	"database/sql"
	"errors"

	"caesar/internal/domain"
	"caesar/internal/infra/db"

	"github.com/google/uuid"
)

type PgClassStore struct {
	db *sql.DB
}

func NewPgClassStore(database *sql.DB) *PgClassStore {
	return &PgClassStore{db: database}
}

func (s *PgClassStore) Create(ctx context.Context, ownerID uuid.UUID, title string) (uuid.UUID, error) {
	exec := db.ExecutorFromContext(ctx, s.db)
	var id uuid.UUID
	err := exec.QueryRowContext(ctx,
		`INSERT INTO class (title, owner_id) VALUES ($1, $2) RETURNING id`,
		title, ownerID,
	).Scan(&id)
	return id, err
}

func (s *PgClassStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.ClassListItem, error) {
	exec := db.ExecutorFromContext(ctx, s.db)
	var item domain.ClassListItem
	err := exec.QueryRowContext(ctx, `
		SELECT c.id, c.title, c.owner_id, c.student_count,
		       o.name || ' ' || o.surname AS owner_name
		FROM class c
		JOIN "user" o ON o.id = c.owner_id
		WHERE c.id = $1
	`, id).Scan(&item.ID, &item.Title, &item.OwnerID, &item.StudentCount, &item.OwnerName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &item, err
}

// ListByUserID возвращает классы пользователя с фильтрацией по его роли.
// role=teacher: классы, где пользователь — владелец или учитель (owner первыми).
// role=student: классы, где пользователь — студент.
func (s *PgClassStore) ListByUserID(ctx context.Context, userID uuid.UUID, role domain.ClassMemberRole) ([]domain.ClassListItem, error) {
	var rows *sql.Rows
	var err error

	switch role {
	case domain.ClassMemberRoleTeacher:
		rows, err = s.db.QueryContext(ctx, `
			SELECT c.id, c.title, c.owner_id, c.student_count,
			       o.name || ' ' || o.surname AS owner_name
			FROM class c
			JOIN "user" o ON o.id = c.owner_id
			LEFT JOIN class_member cm
			       ON cm.class_id = c.id
			      AND cm.user_id  = $1
			      AND cm.role     = $2
			WHERE c.owner_id = $1 OR cm.user_id IS NOT NULL
			ORDER BY
			    CASE WHEN c.owner_id = $1 THEN 0 ELSE 1 END,
			    c.created_at DESC
		`, userID, domain.ClassMemberRoleTeacher)

	case domain.ClassMemberRoleStudent:
		rows, err = s.db.QueryContext(ctx, `
			SELECT c.id, c.title, c.owner_id, c.student_count,
			       o.name || ' ' || o.surname AS owner_name
			FROM class c
			JOIN "user" o ON o.id = c.owner_id
			JOIN class_member cm
			  ON cm.class_id = c.id
			 AND cm.user_id  = $1
			 AND cm.role     = $2
			ORDER BY c.created_at DESC
		`, userID, domain.ClassMemberRoleStudent)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var classes []domain.ClassListItem
	for rows.Next() {
		var item domain.ClassListItem
		if err := rows.Scan(&item.ID, &item.Title, &item.OwnerID, &item.StudentCount, &item.OwnerName); err != nil {
			return nil, err
		}
		item.IsOwner = item.OwnerID == userID
		classes = append(classes, item)
	}
	return classes, rows.Err()
}

func (s *PgClassStore) Update(ctx context.Context, id uuid.UUID, title string) error {
	exec := db.ExecutorFromContext(ctx, s.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE class SET title = $1 WHERE id = $2`,
		title, id,
	)
	return err
}

func (s *PgClassStore) Delete(ctx context.Context, id uuid.UUID) error {
	exec := db.ExecutorFromContext(ctx, s.db)
	_, err := exec.ExecContext(ctx, `DELETE FROM class WHERE id = $1`, id)
	return err
}
