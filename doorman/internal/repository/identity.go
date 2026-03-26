package repository

import (
	"context"
	"database/sql"
	"doorman/internal/domain"
	"doorman/internal/infra/db"
	"errors"
)

type PgIdentityStore struct {
	db *sql.DB
}

func NewPgIdentityStore(db *sql.DB) *PgIdentityStore {
	return &PgIdentityStore{db: db}
}

func (s *PgIdentityStore) GetByPhone(ctx context.Context, phone string) (*domain.Identity, error) {
	exec := db.ExecutorFromContext(ctx, s.db)

	row := exec.QueryRowContext(ctx,
		`SELECT id, phone, status FROM identity WHERE phone = $1`,
		phone,
	)

	identity := &domain.Identity{}
	err := row.Scan(&identity.ID, &identity.Phone, &identity.Status)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return identity, nil
}

func (s *PgIdentityStore) UpdateStatus(ctx context.Context, userID string, status domain.IdentityStatus) error {
	exec := db.ExecutorFromContext(ctx, s.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE identity SET status = $1 WHERE id = $2`,
		status, userID,
	)
	return err
}

func (s *PgIdentityStore) Create(ctx context.Context, phone string) (domain.Identity, error) {
	exec := db.ExecutorFromContext(ctx, s.db)

	row := exec.QueryRowContext(ctx,
		`INSERT INTO identity (phone) VALUES ($1) RETURNING id, phone, status`,
		phone,
	)

	var identity domain.Identity
	if err := row.Scan(&identity.ID, &identity.Phone, &identity.Status); err != nil {
		return domain.Identity{}, err
	}

	return identity, nil
}
