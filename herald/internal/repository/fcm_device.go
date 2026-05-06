package repository

import (
	"context"
	"database/sql"
	"fmt"
	"herald/internal/domain"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type PgFCMDeviceRepository struct {
	db *sql.DB
}

func NewPgFCMDeviceRepository(db *sql.DB) *PgFCMDeviceRepository {
	return &PgFCMDeviceRepository{db: db}
}

func (r *PgFCMDeviceRepository) Upsert(ctx context.Context, userID uuid.UUID, fcmToken, platform string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO fcm_device (user_id, fcm_token, platform)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (fcm_token) DO UPDATE SET user_id = $1, platform = $3`,
		userID, fcmToken, platform,
	)
	if err != nil {
		return fmt.Errorf("FCMDevice.Upsert: %w", err)
	}
	return nil
}

func (r *PgFCMDeviceRepository) Delete(ctx context.Context, fcmToken string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM fcm_device WHERE fcm_token = $1`,
		fcmToken,
	)
	if err != nil {
		return fmt.Errorf("FCMDevice.Delete: %w", err)
	}
	return nil
}

func (r *PgFCMDeviceRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.FCMDevice, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, fcm_token, platform, created_at
		 FROM fcm_device WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("FCMDevice.ListByUserID: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var devices []domain.FCMDevice
	for rows.Next() {
		var d domain.FCMDevice
		if err := rows.Scan(&d.ID, &d.UserID, &d.FCMToken, &d.Platform, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("FCMDevice.ListByUserID scan: %w", err)
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

func (r *PgFCMDeviceRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM fcm_device WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("FCMDevice.DeleteByUserID: %w", err)
	}
	return nil
}

func (r *PgFCMDeviceRepository) DeleteTokens(ctx context.Context, tokens []string) error {
	if len(tokens) == 0 {
		return nil
	}
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM fcm_device WHERE fcm_token = ANY($1)`,
		pq.Array(tokens),
	)
	if err != nil {
		return fmt.Errorf("FCMDevice.DeleteTokens: %w", err)
	}
	return nil
}
