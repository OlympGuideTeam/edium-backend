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

func (r *PgQuizRepository) Create(ctx context.Context, authorID uuid.UUID, title string, description *string, settings domain.QuizDefaultSettings, source domain.QuizSource, courseID *uuid.UUID) (uuid.UUID, error) {
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal settings: %w", err)
	}

	exec := db.ExecutorFromContext(ctx, r.db)

	var id uuid.UUID
	err = exec.QueryRowContext(ctx,
		`INSERT INTO quiz_template (author_id, title, description, default_settings, source, course_id)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		authorID, title, description, settingsJSON, source, courseID,
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
		`SELECT id, author_id, title, description, default_settings, is_public, source, need_evaluation, question_count, course_id, library_session_id, created_at, updated_at
		 FROM quiz_template WHERE id = $1`,
		id,
	).Scan(&q.ID, &q.AuthorID, &q.Title, &q.Description, &settingsJSON,
		&q.IsPublic, &q.Source, &q.NeedEvaluation, &q.QuestionCount, &q.CourseID, &q.LibrarySessionID, &q.CreatedAt, &q.UpdatedAt)
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

func (r *PgQuizRepository) Delete(ctx context.Context, id uuid.UUID) error {
	exec := db.ExecutorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx, `DELETE FROM quiz_template WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete quiz_template: %w", err)
	}
	return nil
}

func (r *PgQuizRepository) SetLibrarySession(ctx context.Context, quizID, sessionID uuid.UUID) error {
	exec := db.ExecutorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE quiz_template SET library_session_id = $1 WHERE id = $2`,
		sessionID, quizID,
	)
	if err != nil {
		return fmt.Errorf("set library_session_id: %w", err)
	}
	return nil
}

func (r *PgQuizRepository) Update(ctx context.Context, id uuid.UUID, title, description *string, settings *domain.QuizDefaultSettings) error {
	var settingsJSON any
	if settings != nil {
		b, err := json.Marshal(*settings)
		if err != nil {
			return fmt.Errorf("marshal settings: %w", err)
		}
		settingsJSON = b
	}

	exec := db.ExecutorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE quiz_template
		 SET title            = COALESCE($1, title),
		     description      = COALESCE($2, description),
		     default_settings = COALESCE($3, default_settings)
		 WHERE id = $4`,
		title, description, settingsJSON, id,
	)
	if err != nil {
		return fmt.Errorf("update quiz_template: %w", err)
	}
	return nil
}

func (r *PgQuizRepository) Publish(ctx context.Context, id uuid.UUID) error {
	exec := db.ExecutorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE quiz_template SET is_public = true WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("publish quiz_template: %w", err)
	}
	return nil
}

func (r *PgQuizRepository) Copy(ctx context.Context, sourceID, newAuthorID uuid.UUID, source domain.QuizSource, courseID *uuid.UUID) (uuid.UUID, error) {
	exec := db.ExecutorFromContext(ctx, r.db)

	var newID uuid.UUID
	err := exec.QueryRowContext(ctx,
		`INSERT INTO quiz_template (author_id, title, description, default_settings, need_evaluation, source, course_id)
		 SELECT $1, title, description, default_settings, need_evaluation, $3, $4
		 FROM quiz_template WHERE id = $2
		 RETURNING id`,
		newAuthorID, sourceID, source, courseID,
	).Scan(&newID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("copy quiz_template: %w", err)
	}

	_, err = exec.ExecContext(ctx, `
		WITH old_questions AS (
			SELECT id, order_index FROM question WHERE quiz_template_id = $1
		),
		new_questions AS (
			INSERT INTO question (quiz_template_id, type, text, image_id, order_index, metadata, max_score)
			SELECT $2, type, text, image_id, order_index, metadata, max_score
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

	return newID, nil
}

func (r *PgQuizRepository) ListPublished(ctx context.Context, needEvaluationFalseOnly bool) ([]domain.QuizListItem, error) {
	exec := db.ExecutorFromContext(ctx, r.db)

	query := `SELECT id, title, description, default_settings, is_public, source, need_evaluation, question_count
	          FROM quiz_template
	          WHERE is_public = true`
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
		`SELECT id, title, description, default_settings, is_public, source, need_evaluation, question_count
		 FROM quiz_template
		 WHERE author_id = $1 AND source = 'library'
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
			&settingsJSON, &item.IsPublic, &item.Source, &item.NeedEvaluation, &item.QuestionCount); err != nil {
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

func (r *PgQuizRepository) GetQuestionByID(ctx context.Context, id uuid.UUID) (*domain.Question, error) {
	exec := db.ExecutorFromContext(ctx, r.db)
	var q domain.Question
	var metaJSON []byte
	err := exec.QueryRowContext(ctx,
		`SELECT id, quiz_template_id, type, text, image_id, order_index, metadata, max_score, created_at, updated_at
		 FROM question WHERE id = $1`,
		id,
	).Scan(&q.ID, &q.QuizTemplateID, &q.Type, &q.Text, &q.ImageID,
		&q.OrderIndex, &metaJSON, &q.MaxScore, &q.CreatedAt, &q.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get question: %w", err)
	}
	if err := json.Unmarshal(metaJSON, &q.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	return &q, nil
}

func (r *PgQuizRepository) SumMaxScore(ctx context.Context, quizTemplateID uuid.UUID) (int, error) {
	var sum int
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(max_score), 0) FROM question WHERE quiz_template_id = $1`,
		quizTemplateID,
	).Scan(&sum)
	if err != nil {
		return 0, fmt.Errorf("sum max score: %w", err)
	}
	return sum, nil
}

func nullableBytes(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}
