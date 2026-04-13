package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"riddler/internal/domain"
	"riddler/internal/infra/db"
)

const claimPendingQuery = `
WITH claimed AS (
    SELECT id
    FROM task
    WHERE task_type = $1
      AND status    = $2
      AND available_at <= NOW()
    ORDER BY available_at
    FOR UPDATE SKIP LOCKED
    LIMIT $3
)
UPDATE task
SET status     = $4,
    attempts   = attempts + 1,
    updated_at = NOW()
FROM claimed
WHERE task.id = claimed.id
RETURNING task.id, task.task_type, task.payload, task.status,
          task.attempts, task.available_at, task.max_attempts, task.trace_ctx`

const markDoneQuery = `
UPDATE task
SET status = $1, updated_at = NOW()
WHERE id = $2`

const markFailedQuery = `
UPDATE task
SET status       = CASE WHEN attempts >= max_attempts THEN $1::task_status ELSE $2::task_status END,
    last_error   = $3,
    available_at = NOW() + $4,
    updated_at   = NOW()
WHERE id = $5`

type PgTaskRepository struct {
	db *sql.DB
}

func NewPgTaskRepository(database *sql.DB) *PgTaskRepository {
	return &PgTaskRepository{db: database}
}

func (r *PgTaskRepository) Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error {
	m := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, m)
	traceCtx := m.Get("traceparent")

	executor := db.ExecutorFromContext(ctx, r.db)
	_, err := executor.ExecContext(ctx,
		`INSERT INTO task (task_type, payload, trace_ctx) VALUES ($1, $2, $3)`,
		taskType, payload, nullableString(traceCtx),
	)
	return err
}

func (r *PgTaskRepository) ClaimPending(ctx context.Context, taskType domain.TaskType, limit int) ([]domain.Task, error) {
	rows, err := r.db.QueryContext(ctx, claimPendingQuery,
		taskType, domain.TaskStatusPending, limit, domain.TaskStatusProcessing,
	)
	if err != nil {
		return nil, fmt.Errorf("ClaimPending: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tasks []domain.Task
	for rows.Next() {
		var t domain.Task
		var maxAttempts int64
		var traceCtx sql.NullString
		if err := rows.Scan(
			&t.ID, &t.Type, &t.Payload, &t.Status,
			&t.Attempts, &t.AvailableAt, &maxAttempts, &traceCtx,
		); err != nil {
			return nil, fmt.Errorf("ClaimPending scan: %w", err)
		}
		t.MaxAttempts = &maxAttempts
		t.TraceCtx = traceCtx.String
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (r *PgTaskRepository) MarkDone(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, markDoneQuery, domain.TaskStatusDone, id)
	if err != nil {
		return fmt.Errorf("MarkDone: %w", err)
	}
	return nil
}

func (r *PgTaskRepository) MarkFailed(ctx context.Context, id uuid.UUID, reason string, retryAfter time.Duration) error {
	_, err := r.db.ExecContext(ctx, markFailedQuery,
		domain.TaskStatusFailed, domain.TaskStatusPending, reason, retryAfter, id,
	)
	if err != nil {
		return fmt.Errorf("MarkFailed: %w", err)
	}
	return nil
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
