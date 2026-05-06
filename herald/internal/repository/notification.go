package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"herald/internal/domain"
	"herald/internal/pkg/apperr"

	"github.com/google/uuid"
)

type PgNotificationRepository struct {
	db *sql.DB
}

func NewPgNotificationRepository(db *sql.DB) *PgNotificationRepository {
	return &PgNotificationRepository{db: db}
}

func (r *PgNotificationRepository) Save(ctx context.Context, n *domain.Notification) error {
	var dataJSON []byte
	if n.Data != nil {
		var err error
		dataJSON, err = json.Marshal(n.Data)
		if err != nil {
			return fmt.Errorf("Notification.Save marshal data: %w", err)
		}
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO notification (id, user_id, title, body, data)
		 VALUES ($1, $2, $3, $4, $5)`,
		n.ID, n.UserID, n.Title, n.Body, nullableBytes(dataJSON),
	)
	if err != nil {
		return fmt.Errorf("Notification.Save: %w", err)
	}
	return nil
}

func (r *PgNotificationRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Notification, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, title, body, is_read, data, created_at
		 FROM notification
		 WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT 100`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("Notification.ListByUserID: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var notifications []domain.Notification
	for rows.Next() {
		var n domain.Notification
		var dataJSON sql.NullString
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Body, &n.IsRead, &dataJSON, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("Notification.ListByUserID scan: %w", err)
		}
		if dataJSON.Valid && dataJSON.String != "" && dataJSON.String != "null" {
			var d domain.NotificationData
			if err := json.Unmarshal([]byte(dataJSON.String), &d); err != nil {
				return nil, fmt.Errorf("Notification.ListByUserID unmarshal data: %w", err)
			}
			n.Data = &d
		}
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

func (r *PgNotificationRepository) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE notification SET is_read = true WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("Notification.MarkRead: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Notification.MarkRead rows: %w", err)
	}
	if n == 0 {
		return apperr.ErrNotificationNotFound
	}
	return nil
}

func (r *PgNotificationRepository) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notification WHERE user_id = $1 AND is_read = false`,
		userID,
	).Scan(&count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("Notification.CountUnread: %w", err)
	}
	return count, nil
}

func nullableBytes(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}
