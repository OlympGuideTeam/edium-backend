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

type PgAttemptRepository struct {
	db *sql.DB
}

func NewPgAttemptRepository(database *sql.DB) *PgAttemptRepository {
	return &PgAttemptRepository{db: database}
}

func (r *PgAttemptRepository) Create(ctx context.Context, sessionID, userID uuid.UUID, questionOrder []uuid.UUID) (uuid.UUID, error) {
	orderJSON, err := json.Marshal(questionOrder)
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal question_order: %w", err)
	}

	exec := db.ExecutorFromContext(ctx, r.db)

	var id uuid.UUID
	err = exec.QueryRowContext(ctx,
		`INSERT INTO attempt (session_id, user_id, question_order)
		 VALUES ($1, $2, $3)
		 RETURNING id`,
		sessionID, userID, orderJSON,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert attempt: %w", err)
	}
	return id, nil
}

func (r *PgAttemptRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Attempt, error) {
	exec := db.ExecutorFromContext(ctx, r.db)

	var a domain.Attempt
	var orderJSON []byte
	err := exec.QueryRowContext(ctx,
		`SELECT id, session_id, user_id, status, score, grade, question_order, started_at, finished_at
		 FROM attempt WHERE id = $1`,
		id,
	).Scan(&a.ID, &a.SessionID, &a.UserID, &a.Status, &a.Score, &a.Grade, &orderJSON, &a.StartedAt, &a.FinishedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get attempt: %w", err)
	}
	if err := json.Unmarshal(orderJSON, &a.QuestionOrder); err != nil {
		return nil, fmt.Errorf("unmarshal question_order: %w", err)
	}
	return &a, nil
}

func (r *PgAttemptRepository) SetGrading(ctx context.Context, attemptID uuid.UUID) error {
	exec := db.ExecutorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE attempt SET status = $2, finished_at = now(), updated_at = now() WHERE id = $1`,
		attemptID, domain.AttemptStatusGrading,
	)
	if err != nil {
		return fmt.Errorf("set grading: %w", err)
	}
	return nil
}

func (r *PgAttemptRepository) SetGraded(ctx context.Context, attemptID uuid.UUID, score float64) error {
	exec := db.ExecutorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE attempt SET status = $2, score = $3, updated_at = now() WHERE id = $1`,
		attemptID, domain.AttemptStatusGraded, score,
	)
	if err != nil {
		return fmt.Errorf("set graded: %w", err)
	}
	return nil
}

func (r *PgAttemptRepository) Complete(ctx context.Context, attemptID uuid.UUID, score, grade float64) error {
	exec := db.ExecutorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE attempt
		 SET status = $2, score = $3, grade = $4, finished_at = now(), updated_at = now()
		 WHERE id = $1`,
		attemptID, domain.AttemptStatusCompleted, score, grade,
	)
	if err != nil {
		return fmt.Errorf("complete attempt: %w", err)
	}
	return nil
}

func (r *PgAttemptRepository) CompleteGraded(ctx context.Context, attemptID uuid.UUID, score, grade float64) error {
	exec := db.ExecutorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE attempt SET status = $2, score = $3, grade = $4, updated_at = now() WHERE id = $1`,
		attemptID, domain.AttemptStatusCompleted, score, grade,
	)
	if err != nil {
		return fmt.Errorf("complete graded: %w", err)
	}
	return nil
}

func (r *PgAttemptRepository) UpdateScore(ctx context.Context, attemptID uuid.UUID, score float64) error {
	exec := db.ExecutorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE attempt SET score = $2, updated_at = now() WHERE id = $1`,
		attemptID, score,
	)
	if err != nil {
		return fmt.Errorf("update attempt score: %w", err)
	}
	return nil
}

func (r *PgAttemptRepository) FindBySessionID(ctx context.Context, sessionID uuid.UUID) ([]domain.AttemptSummary, error) {
	exec := db.ExecutorFromContext(ctx, r.db)
	rows, err := exec.QueryContext(ctx,
		`SELECT id, user_id, status, score FROM attempt WHERE session_id = $1 ORDER BY started_at`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("find attempts by session: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.AttemptSummary
	for rows.Next() {
		var a domain.AttemptSummary
		if err := rows.Scan(&a.ID, &a.UserID, &a.Status, &a.Score); err != nil {
			return nil, fmt.Errorf("scan attempt summary: %w", err)
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (r *PgAttemptRepository) GetUserStatistic(ctx context.Context, userID uuid.UUID) (*domain.UserStatistic, error) {
	var st domain.UserStatistic
	err := r.db.QueryRowContext(ctx, `
		SELECT
		    COUNT(*) FILTER (WHERE status = 'completed'),
		    COALESCE(AVG(grade) FILTER (WHERE status = 'completed' AND grade IS NOT NULL), 0),
		    (SELECT COUNT(*) FROM quiz_session qs
		     JOIN quiz_template qt ON qt.id = qs.quiz_template_id
		     WHERE qt.author_id = $1 AND qs.status = 'finished')
		FROM attempt
		WHERE user_id = $1
	`, userID).Scan(&st.QuizCountPassed, &st.AvgQuizScore, &st.QuizSessionsConducted)
	if err != nil {
		return nil, fmt.Errorf("get user statistic: %w", err)
	}
	return &st, nil
}

func (r *PgAttemptRepository) FindExpiredInProgress(ctx context.Context) ([]domain.Attempt, error) {
	exec := db.ExecutorFromContext(ctx, r.db)

	rows, err := exec.QueryContext(ctx,
		`SELECT a.id, a.session_id, a.user_id, a.status, a.score, a.question_order, a.started_at, a.finished_at
		 FROM attempt a
		 JOIN quiz_session qs ON qs.id = a.session_id
		 WHERE a.status = $1
		   AND qs.total_time_limit_sec IS NOT NULL
		   AND a.started_at + (qs.total_time_limit_sec * interval '1 second') < now()`,
		domain.AttemptStatusInProgress,
	)
	if err != nil {
		return nil, fmt.Errorf("query expired attempts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.Attempt
	for rows.Next() {
		var a domain.Attempt
		var orderJSON []byte
		if err := rows.Scan(&a.ID, &a.SessionID, &a.UserID, &a.Status, &a.Score,
			&orderJSON, &a.StartedAt, &a.FinishedAt); err != nil {
			return nil, fmt.Errorf("scan attempt: %w", err)
		}
		if len(orderJSON) > 0 {
			if err := json.Unmarshal(orderJSON, &a.QuestionOrder); err != nil {
				return nil, fmt.Errorf("unmarshal question_order: %w", err)
			}
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
