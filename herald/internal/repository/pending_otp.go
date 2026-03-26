package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"herald/internal/domain"
	"herald/internal/infra/db"
)

type PgPendingOTPRepository struct {
	db *sql.DB
}

func NewPgPendingOTPRepository(database *sql.DB) *PgPendingOTPRepository {
	return &PgPendingOTPRepository{db: database}
}

func (r *PgPendingOTPRepository) Save(ctx context.Context, phone string, chatID int64) error {
	executor := db.ExecutorFromContext(ctx, r.db)
	_, err := executor.ExecContext(ctx,
		`INSERT INTO pending_otp (phone, chat_id) VALUES ($1, $2) ON CONFLICT (phone) DO UPDATE SET chat_id = $2, expires_at = NOW() + INTERVAL '10 minutes'`,
		phone, chatID,
	)
	if err != nil {
		return fmt.Errorf("PendingOTP.Save: %w", err)
	}
	return nil
}

func (r *PgPendingOTPRepository) Get(ctx context.Context, phone string) (*domain.PendingOTP, error) {
	executor := db.ExecutorFromContext(ctx, r.db)
	row := executor.QueryRowContext(ctx,
		`SELECT phone, chat_id, expires_at FROM pending_otp WHERE phone = $1 AND expires_at > NOW()`,
		phone,
	)

	var p domain.PendingOTP
	if err := row.Scan(&p.Phone, &p.ChatID, &p.ExpiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("PendingOTP.Get: %w", err)
	}
	return &p, nil
}

func (r *PgPendingOTPRepository) Delete(ctx context.Context, phone string) error {
	executor := db.ExecutorFromContext(ctx, r.db)
	_, err := executor.ExecContext(ctx, `DELETE FROM pending_otp WHERE phone = $1`, phone)
	if err != nil {
		return fmt.Errorf("PendingOTP.Delete: %w", err)
	}
	return nil
}
