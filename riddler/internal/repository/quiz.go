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

func (r *PgQuizRepository) Update(ctx context.Context, id uuid.UUID, title, description *string) error {
	exec := db.ExecutorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE quiz_template
		 SET title       = COALESCE($1, title),
		     description = COALESCE($2, description)
		 WHERE id = $3`,
		title, description, id,
	)
	if err != nil {
		return fmt.Errorf("update quiz_template: %w", err)
	}
	return nil
}

func (r *PgQuizRepository) Publish(ctx context.Context, id uuid.UUID, isPublic bool) error {
	exec := db.ExecutorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE quiz_template SET is_draft = false, is_public = $1 WHERE id = $2`,
		isPublic, id,
	)
	if err != nil {
		return fmt.Errorf("publish quiz_template: %w", err)
	}
	return nil
}

func (r *PgQuizRepository) Copy(ctx context.Context, sourceID, newAuthorID uuid.UUID) (uuid.UUID, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var newID uuid.UUID
	err = tx.QueryRowContext(ctx,
		`INSERT INTO quiz_template (author_id, title, description, default_settings, need_evaluation)
		 SELECT $1, title, description, default_settings, need_evaluation
		 FROM quiz_template WHERE id = $2
		 RETURNING id`,
		newAuthorID, sourceID,
	).Scan(&newID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("copy quiz_template: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		WITH old_questions AS (
			SELECT id, order_index FROM question WHERE quiz_template_id = $1
		),
		new_questions AS (
			INSERT INTO question (quiz_template_id, type, text, image_link, order_index, metadata, max_score)
			SELECT $2, type, text, image_link, order_index, metadata, max_score
			FROM question WHERE quiz_template_id = $1
			RETURNING id, order_index
		),
		question_mapping AS (
			SELECT old_questions.id AS old_id, new_questions.id AS new_id
			FROM old_questions JOIN new_questions USING (order_index)
		)
		INSERT INTO answer_option (question_id, text, is_correct)
		SELECT qm.new_id, ao.text, ao.is_correct
		FROM answer_option ao
		JOIN question_mapping qm ON ao.question_id = qm.old_id`,
		sourceID, newID,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("copy questions and options: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return uuid.Nil, fmt.Errorf("commit: %w", err)
	}

	return newID, nil
}

func (r *PgQuizRepository) ListPublished(ctx context.Context, needEvaluationFalseOnly bool) ([]domain.QuizListItem, error) {
	exec := db.ExecutorFromContext(ctx, r.db)

	query := `SELECT id, title, description, default_settings, is_public, is_draft, need_evaluation, question_count
	          FROM quiz_template
	          WHERE is_draft = false`
	if needEvaluationFalseOnly {
		query += ` AND need_evaluation = false`
	}
	query += ` ORDER BY created_at DESC`

	rows, err := exec.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list published quizzes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanQuizListItems(rows)
}

func (r *PgQuizRepository) ListByAuthor(ctx context.Context, authorID uuid.UUID) ([]domain.QuizListItem, error) {
	exec := db.ExecutorFromContext(ctx, r.db)

	rows, err := exec.QueryContext(ctx,
		`SELECT id, title, description, default_settings, is_public, is_draft, need_evaluation, question_count
		 FROM quiz_template
		 WHERE author_id = $1
		 ORDER BY created_at DESC`,
		authorID,
	)
	if err != nil {
		return nil, fmt.Errorf("list quizzes by author: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanQuizListItems(rows)
}

func scanQuizListItems(rows *sql.Rows) ([]domain.QuizListItem, error) {
	var result []domain.QuizListItem
	for rows.Next() {
		var item domain.QuizListItem
		var settingsJSON []byte
		if err := rows.Scan(&item.ID, &item.Title, &item.Description,
			&settingsJSON, &item.IsPublic, &item.IsDraft, &item.NeedEvaluation, &item.QuestionCount); err != nil {
			return nil, fmt.Errorf("scan quiz list item: %w", err)
		}
		if err := json.Unmarshal(settingsJSON, &item.DefaultSettings); err != nil {
			return nil, fmt.Errorf("unmarshal settings: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("quiz list rows: %w", err)
	}
	return result, nil
}

func nullableBytes(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}
