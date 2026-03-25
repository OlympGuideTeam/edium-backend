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

func (r *PgPendingOTPRepository) Save(ctx context.Context, correlationID string, chatID int64) error {
	executor := db.ExecutorFromContext(ctx, r.db)
	_, err := executor.ExecContext(ctx,
		`INSERT INTO pending_otp (correlation_id, chat_id) VALUES ($1, $2)`,
		correlationID, chatID,
	)
	if err != nil {
		return fmt.Errorf("PendingOTP.Save: %w", err)
	}
	return nil
}

func (r *PgPendingOTPRepository) Get(ctx context.Context, correlationID string) (*domain.PendingOTP, error) {
	executor := db.ExecutorFromContext(ctx, r.db)
	row := executor.QueryRowContext(ctx,
		`SELECT correlation_id, chat_id, expires_at
         FROM pending_otp
         WHERE correlation_id = $1 AND expires_at > NOW()`,
		correlationID,
	)

	var p domain.PendingOTP
	if err := row.Scan(&p.CorrelationID, &p.ChatID, &p.ExpiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("PendingOTP.Get: %w", err)
	}
	return &p, nil
}

func (r *PgPendingOTPRepository) Delete(ctx context.Context, correlationID string) error {
	executor := db.ExecutorFromContext(ctx, r.db)
	_, err := executor.ExecContext(ctx,
		`DELETE FROM pending_otp WHERE correlation_id = $1`, correlationID,
	)
	if err != nil {
		return fmt.Errorf("PendingOTP.Delete: %w", err)
	}
	return nil
}
