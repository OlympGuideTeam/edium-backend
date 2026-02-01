package otp

import (
	"context"
	"database/sql"
	"doorman/internal/domain"
)

type IdentityRepo struct {
	db *sql.DB
}

func (r *IdentityRepo) GetByPhone(ctx context.Context, phone string) (*domain.Identity, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, phone, status FROM identity WHERE phone = $1`,
		phone,
	)

	if row.Err() != nil {
		return nil, row.Err()
	}

	identity := &domain.Identity{}
	err := row.Scan(identity.ID, &identity.Phone, &identity.Status)
	if err != nil {
		return nil, err
	}

	return identity, nil
}

func (r *IdentityRepo) Create(ctx context.Context, identity *domain.Identity) error {
	return nil // TODO
}
