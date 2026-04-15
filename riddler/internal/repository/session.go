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

type PgSessionRepository struct {
	db *sql.DB
}

func NewPgSessionRepository(database *sql.DB) *PgSessionRepository {
	return &PgSessionRepository{db: database}
}

func (r *PgSessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.QuizSession, error) {
	exec := db.ExecutorFromContext(ctx, r.db)

	var s domain.QuizSession
	err := exec.QueryRowContext(ctx,
		`SELECT id, quiz_template_id, mode, status, total_time_limit_sec, question_time_limit_sec,
		        shuffle_questions, started_at, finished_at
		 FROM quiz_session WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.QuizTemplateID, &s.Mode, &s.Status,
		&s.TotalTimeLimitSec, &s.QuestionTimeLimitSec, &s.ShuffleQuestions,
		&s.StartedAt, &s.FinishedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session by id: %w", err)
	}
	return &s, nil
}

func (r *PgSessionRepository) Create(ctx context.Context, p domain.CreateSessionParams) (uuid.UUID, error) {
	var settingsJSON []byte
	if p.Settings != nil {
		var err error
		settingsJSON, err = json.Marshal(p.Settings)
		if err != nil {
			return uuid.Nil, fmt.Errorf("marshal settings: %w", err)
		}
	}

	exec := db.ExecutorFromContext(ctx, r.db)

	var id uuid.UUID
	err := exec.QueryRowContext(ctx,
		`INSERT INTO quiz_session
		 (quiz_template_id, mode, total_time_limit_sec, question_time_limit_sec,
		  shuffle_questions, status, settings, started_at, finished_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE($8, now()), $9)
		 RETURNING id`,
		p.QuizTemplateID,
		p.Mode,
		p.TotalTimeLimitSec,
		p.QuestionTimeLimitSec,
		p.ShuffleQuestions,
		p.Status,
		settingsJSON,
		p.StartedAt,
		p.FinishedAt,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert quiz_session: %w", err)
	}
	return id, nil
}
