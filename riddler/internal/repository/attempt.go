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
		`SELECT id, session_id, user_id, status, score, question_order, started_at, finished_at
		 FROM attempt WHERE id = $1`,
		id,
	).Scan(&a.ID, &a.SessionID, &a.UserID, &a.Status, &a.Score, &orderJSON, &a.StartedAt, &a.FinishedAt)
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

func (r *PgAttemptRepository) UpsertAnswer(ctx context.Context, attemptID, questionID uuid.UUID, answerData map[string]any) (uuid.UUID, error) {
	dataJSON, err := json.Marshal(answerData)
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal answer_data: %w", err)
	}

	exec := db.ExecutorFromContext(ctx, r.db)

	var id uuid.UUID
	err = exec.QueryRowContext(ctx,
		`INSERT INTO answer_submission (attempt_id, question_id, answer_data)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (attempt_id, question_id)
		 DO UPDATE SET answer_data = EXCLUDED.answer_data, updated_at = now()
		 RETURNING id`,
		attemptID, questionID, dataJSON,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert answer: %w", err)
	}
	return id, nil
}

func (r *PgAttemptRepository) GetAnswers(ctx context.Context, attemptID uuid.UUID) ([]domain.AnswerSubmission, error) {
	exec := db.ExecutorFromContext(ctx, r.db)

	rows, err := exec.QueryContext(ctx,
		`SELECT id, attempt_id, question_id, answer_data, final_score, final_source, final_feedback
		 FROM answer_submission WHERE attempt_id = $1`,
		attemptID,
	)
	if err != nil {
		return nil, fmt.Errorf("query answers: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.AnswerSubmission
	for rows.Next() {
		var s domain.AnswerSubmission
		var dataJSON []byte
		var finalSource sql.NullString
		var finalFeedback sql.NullString
		if err := rows.Scan(&s.ID, &s.AttemptID, &s.QuestionID, &dataJSON,
			&s.FinalScore, &finalSource, &finalFeedback); err != nil {
			return nil, fmt.Errorf("scan answer: %w", err)
		}
		if err := json.Unmarshal(dataJSON, &s.AnswerData); err != nil {
			return nil, fmt.Errorf("unmarshal answer_data: %w", err)
		}
		if finalSource.Valid {
			src := domain.FinalSource(finalSource.String)
			s.FinalSource = &src
		}
		if finalFeedback.Valid {
			s.FinalFeedback = &finalFeedback.String
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *PgAttemptRepository) EvaluateSubmission(ctx context.Context, submissionID uuid.UUID, score float64, source domain.FinalSource, feedback *string) error {
	exec := db.ExecutorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE answer_submission
		 SET final_score = $2, final_source = $3, final_feedback = $4, updated_at = now()
		 WHERE id = $1`,
		submissionID, score, source, feedback,
	)
	if err != nil {
		return fmt.Errorf("update answer evaluation: %w", err)
	}
	return nil
}

func (r *PgAttemptRepository) Complete(ctx context.Context, attemptID uuid.UUID, score float64) error {
	exec := db.ExecutorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE attempt
		 SET status = $2, score = $3, finished_at = now(), updated_at = now()
		 WHERE id = $1`,
		attemptID, domain.AttemptStatusCompleted, score,
	)
	if err != nil {
		return fmt.Errorf("complete attempt: %w", err)
	}
	return nil
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
