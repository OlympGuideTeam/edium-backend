package repository

import (
	"context"
	"database/sql"
	"doorman/internal/domain"
	"doorman/internal/infra/db"
)

type PgScheduler struct {
	db *sql.DB
}

func NewPgScheduler(db *sql.DB) *PgScheduler {
	return &PgScheduler{db: db}
}

func (s *PgScheduler) Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error {
	executor := db.ExecutorFromContext(ctx, s.db)

	_, err := executor.ExecContext(ctx, "INSERT INTO task (task_type, payload) VALUES ($1, $2)", taskType, payload)

	return err
}
