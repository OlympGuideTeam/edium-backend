package repository

import (
	"context"
	"database/sql"
	"fmt"
	"herald/internal/domain"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type PgSMSTaskRepository struct {
	db *sql.DB
}

func NewPgSMSTaskRepository(database *sql.DB) *PgSMSTaskRepository {
	return &PgSMSTaskRepository{db: database}
}

// Create добавляет SMS-задачу в очередь. При конфликте по idempotency_key (повтор
// otp_sent-задачи) молча пропускает вставку — дубль SMS не создаётся.
func (r *PgSMSTaskRepository) Create(ctx context.Context, phone, text string, idempotencyKey uuid.UUID) error {
	m := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, m)
	traceCtx := m.Get("traceparent")

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sms_task (phone, text, idempotency_key, trace_ctx)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (idempotency_key) WHERE idempotency_key IS NOT NULL DO NOTHING`,
		phone, text, idempotencyKey, nullableString(traceCtx),
	)
	if err != nil {
		return fmt.Errorf("SMSTask.Create: %w", err)
	}
	return nil
}

// ListPending атомарно клеймит до limit задач и возвращает их Android-шлюзу.
// Используется FOR UPDATE SKIP LOCKED: параллельные запросы не получают одни задачи.
// Задачи с просроченным клеймом (> 5 минут) доступны для повторного клейма.
func (r *PgSMSTaskRepository) ListPending(ctx context.Context, limit int) ([]domain.SMSTask, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, phone, text, created_at, processed_at
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
		if err := rows.Scan(&t.ID, &t.Phone, &t.Text, &t.CreatedAt, &t.ProcessedAt); err != nil {
			return nil, fmt.Errorf("SMSTask.ListPending scan: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// Ack подтверждает результат отправки SMS.
// success=true → статус sent.
// success=false → если попытки не исчерпаны, возвращает задачу в очередь (retry);
// иначе — failed.
func (r *PgSMSTaskRepository) Ack(ctx context.Context, id uuid.UUID, success bool, errMsg string) error {
	if success {
		_, err := r.db.ExecContext(ctx,
			`UPDATE sms_task
			 SET status = 'sent', processed_at = NOW()
			 WHERE id = $1 AND status = 'pending'`,
			id,
		)
		if err != nil {
			return fmt.Errorf("SMSTask.Ack: %w", err)
		}
		return nil
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE sms_task
		SET retry_count  = retry_count + 1,
		    status       = CASE WHEN retry_count + 1 >= max_retries THEN 'failed' ELSE 'pending' END,
		    processed_at = CASE WHEN retry_count + 1 >= max_retries THEN NOW() ELSE NULL END
		WHERE id = $1 AND status = 'pending'`,
		id,
	)
	if err != nil {
		return fmt.Errorf("SMSTask.Ack: %w", err)
	}
	return nil
}
