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

func (r *PgPendingOTPRepository) Save(ctx context.Context, phone string, channel domain.Channel, chatID int64) error {
	executor := db.ExecutorFromContext(ctx, r.db)
	_, err := executor.ExecContext(ctx,
		`INSERT INTO pending_otp (phone, channel, chat_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (phone, channel) DO UPDATE
		 SET chat_id = $3, expires_at = NOW() + INTERVAL '10 minutes'`,
		phone, string(channel), chatID,
	)
	if err != nil {
		return fmt.Errorf("PendingOTP.Save: %w", err)
	}
	return nil
}

func (r *PgPendingOTPRepository) Get(ctx context.Context, phone string, channel domain.Channel) (*domain.PendingOTP, error) {
	executor := db.ExecutorFromContext(ctx, r.db)
	row := executor.QueryRowContext(ctx,
		`SELECT phone, channel, chat_id, expires_at
		 FROM pending_otp
		 WHERE phone = $1 AND channel = $2 AND expires_at > NOW()`,
		phone, string(channel),
	)

	var p domain.PendingOTP
	if err := row.Scan(&p.Phone, &p.Channel, &p.ChatID, &p.ExpiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("PendingOTP.Get: %w", err)
	}
	return &p, nil
}

func (r *PgPendingOTPRepository) Delete(ctx context.Context, phone string, channel domain.Channel) error {
	executor := db.ExecutorFromContext(ctx, r.db)
	_, err := executor.ExecContext(ctx,
		`DELETE FROM pending_otp WHERE phone = $1 AND channel = $2`,
		phone, string(channel),
	)
	if err != nil {
		return fmt.Errorf("PendingOTP.Delete: %w", err)
	}
	return nil
}
