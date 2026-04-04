package repository

import (
	"caesar/internal/domain"
	"caesar/internal/infra/db"
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

type PgUserStore struct {
	db *sql.DB
}

func NewPgUserStore(database *sql.DB) *PgUserStore {
	return &PgUserStore{db: database}
}

func (s *PgUserStore) Create(ctx context.Context, u domain.User) error {
	exec := db.ExecutorFromContext(ctx, s.db)
	_, err := exec.ExecContext(ctx, `
		INSERT INTO "user" (id, name, surname, phone, status)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING
	`, u.ID, u.Name, u.Surname, u.Phone, u.Status)
	return err
}

func (s *PgUserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	exec := db.ExecutorFromContext(ctx, s.db)
	row := exec.QueryRowContext(ctx,
		`SELECT id, name, surname, phone, status FROM "user" WHERE id = $1`,
		id,
	)

	var u domain.User
	if err := row.Scan(&u.ID, &u.Name, &u.Surname, &u.Phone, &u.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (s *PgUserStore) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.UserStatus) error {
	exec := db.ExecutorFromContext(ctx, s.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE "user" SET status = $1 WHERE id = $2`,
		status, id,
	)
	return err
}
