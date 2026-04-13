package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/infra/db"
)

type PgQuizRepository struct {
	db *sql.DB
}

func NewPgQuizRepository(database *sql.DB) *PgQuizRepository {
	return &PgQuizRepository{db: database}
}

func (r *PgQuizRepository) Create(ctx context.Context, authorID uuid.UUID, title string, description *string, settings domain.QuizDefaultSettings) (uuid.UUID, error) {
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal settings: %w", err)
	}

	exec := db.ExecutorFromContext(ctx, r.db)

	var id uuid.UUID
	err = exec.QueryRowContext(ctx,
		`INSERT INTO quiz_template (author_id, title, description, default_settings)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		authorID, title, description, settingsJSON,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert quiz_template: %w", err)
	}

	return id, nil
}
