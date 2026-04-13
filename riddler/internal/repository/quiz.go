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

func (r *PgQuizRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.QuizTemplate, error) {
	exec := db.ExecutorFromContext(ctx, r.db)

	var q domain.QuizTemplate
	var settingsJSON []byte
	err := exec.QueryRowContext(ctx,
		`SELECT id, author_id, title, description, default_settings, is_public, is_draft, need_evaluation, created_at, updated_at
		 FROM quiz_template WHERE id = $1`,
		id,
	).Scan(&q.ID, &q.AuthorID, &q.Title, &q.Description, &settingsJSON,
		&q.IsPublic, &q.IsDraft, &q.NeedEvaluation, &q.CreatedAt, &q.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select quiz_template: %w", err)
	}
	if err := json.Unmarshal(settingsJSON, &q.DefaultSettings); err != nil {
		return nil, fmt.Errorf("unmarshal settings: %w", err)
	}
	return &q, nil
}

func (r *PgQuizRepository) AddQuestion(ctx context.Context, params domain.AddQuestionParams) (uuid.UUID, int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var orderIndex int
	err = tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(order_index), 0) + 1 FROM question WHERE quiz_template_id = $1`,
		params.QuizTemplateID,
	).Scan(&orderIndex)
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("get order index: %w", err)
	}

	var metadataJSON []byte
	if params.Metadata != nil {
		metadataJSON, err = json.Marshal(params.Metadata)
		if err != nil {
			return uuid.Nil, 0, fmt.Errorf("marshal metadata: %w", err)
		}
	}

	var questionID uuid.UUID
	err = tx.QueryRowContext(ctx,
		`INSERT INTO question (quiz_template_id, type, text, image_link, order_index, metadata, max_score)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		params.QuizTemplateID, params.Type, params.Text, params.ImageLink,
		orderIndex, nullableBytes(metadataJSON), params.MaxScore,
	).Scan(&questionID)
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("insert question: %w", err)
	}

	for _, opt := range params.Options {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO answer_option (question_id, text, is_correct) VALUES ($1, $2, $3)`,
			questionID, opt.Text, opt.IsCorrect,
		)
		if err != nil {
			return uuid.Nil, 0, fmt.Errorf("insert answer_option: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return uuid.Nil, 0, fmt.Errorf("commit: %w", err)
	}

	return questionID, orderIndex, nil
}

func (r *PgQuizRepository) SetNeedEvaluation(ctx context.Context, quizID uuid.UUID, value bool) error {
	exec := db.ExecutorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE quiz_template SET need_evaluation = $1 WHERE id = $2`,
		value, quizID,
	)
	return err
}

func (r *PgQuizRepository) HasFreeAnswerQuestions(ctx context.Context, quizID uuid.UUID) (bool, error) {
	exec := db.ExecutorFromContext(ctx, r.db)
	var count int
	err := exec.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM question WHERE quiz_template_id = $1 AND type = 'with_free_answer'`,
		quizID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("count free_answer questions: %w", err)
	}
	return count > 0, nil
}

func nullableBytes(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
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
