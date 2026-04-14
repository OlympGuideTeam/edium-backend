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
		 (quiz_template_id, mode, course_item_id, total_time_limit_sec, question_time_limit_sec,
		  shuffle_questions, status, settings, finished_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id`,
		p.QuizTemplateID,
		p.Mode,
		p.CourseItemID,
		p.TotalTimeLimitSec,
		p.QuestionTimeLimitSec,
		p.ShuffleQuestions,
		p.Status,
		settingsJSON,
		p.FinishedAt,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert quiz_session: %w", err)
	}
	return id, nil
}
