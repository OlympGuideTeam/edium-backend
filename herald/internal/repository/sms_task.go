package repository

import (
	"context"
	"database/sql"
	"fmt"
	"herald/internal/domain"

	"github.com/google/uuid"
)

type PgSMSTaskRepository struct {
	db *sql.DB
}

func NewPgSMSTaskRepository(database *sql.DB) *PgSMSTaskRepository {
	return &PgSMSTaskRepository{db: database}
}

// Create добавляет новую SMS-задачу в очередь.
func (r *PgSMSTaskRepository) Create(ctx context.Context, phone, text string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sms_task (phone, text) VALUES ($1, $2)`,
		phone, text,
	)
	if err != nil {
		return fmt.Errorf("SMSTask.Create: %w", err)
	}
	return nil
}

// ListPending возвращает до limit задач со статусом 'pending'.
func (r *PgSMSTaskRepository) ListPending(ctx context.Context, limit int) ([]domain.SMSTask, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, phone, text, status, created_at, processed_at
		 FROM sms_task
		 WHERE status = 'pending'
		 ORDER BY created_at
		 LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("SMSTask.ListPending: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tasks []domain.SMSTask
	for rows.Next() {
		var t domain.SMSTask
		if err := rows.Scan(&t.ID, &t.Phone, &t.Text, &t.Status, &t.CreatedAt, &t.ProcessedAt); err != nil {
			return nil, fmt.Errorf("SMSTask.ListPending scan: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// Ack обновляет статус задачи: sent или failed.
func (r *PgSMSTaskRepository) Ack(ctx context.Context, id uuid.UUID, success bool, errMsg string) error {
	status := domain.SMSTaskStatusSent
	if !success {
		status = domain.SMSTaskStatusFailed
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE sms_task
		 SET status = $1, processed_at = NOW()
		 WHERE id = $2 AND status = 'pending'`,
		string(status), id,
	)
	if err != nil {
		return fmt.Errorf("SMSTask.Ack: %w", err)
	}
	return nil
}
