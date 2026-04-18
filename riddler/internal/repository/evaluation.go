package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/infra/db"
)

func (r *PgAttemptRepository) InsertEvaluation(ctx context.Context, submissionID uuid.UUID, status domain.EvaluationStatus, score *float64, source domain.FinalSource, feedback *string) (uuid.UUID, error) {
	exec := db.ExecutorFromContext(ctx, r.db)
	var id uuid.UUID
	err := exec.QueryRowContext(ctx,
		`INSERT INTO answer_evaluation (submission_id, status, score, source, feedback)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		submissionID, status, score, source, feedback,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert evaluation: %w", err)
	}
	return id, nil
}

func (r *PgAttemptRepository) CompleteEvaluation(ctx context.Context, evalID uuid.UUID, score float64, feedback *string) error {
	exec := db.ExecutorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE answer_evaluation SET status = 'completed', score = $2, feedback = $3 WHERE id = $1`,
		evalID, score, feedback,
	)
	if err != nil {
		return fmt.Errorf("complete evaluation: %w", err)
	}
	return nil
}

func (r *PgAttemptRepository) FailEvaluation(ctx context.Context, evalID uuid.UUID, feedback string) error {
	exec := db.ExecutorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE answer_evaluation SET status = 'failed', feedback = $2 WHERE id = $1`,
		evalID, feedback,
	)
	if err != nil {
		return fmt.Errorf("fail evaluation: %w", err)
	}
	return nil
}

func (r *PgAttemptRepository) GetEvaluationByID(ctx context.Context, evalID uuid.UUID) (*domain.AnswerEvaluation, error) {
	exec := db.ExecutorFromContext(ctx, r.db)
	var e domain.AnswerEvaluation
	var feedback sql.NullString
	err := exec.QueryRowContext(ctx,
		`SELECT ae.id, ae.submission_id, s.attempt_id, s.question_id, ae.status, ae.score, ae.source, ae.feedback
		 FROM answer_evaluation ae
		 JOIN answer_submission s ON s.id = ae.submission_id
		 WHERE ae.id = $1`,
		evalID,
	).Scan(&e.ID, &e.SubmissionID, &e.AttemptID, &e.QuestionID, &e.Status, &e.Score, &e.Source, &feedback)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get evaluation: %w", err)
	}
	if feedback.Valid {
		e.Feedback = &feedback.String
	}
	return &e, nil
}

func (r *PgAttemptRepository) HasPendingLLMEvaluations(ctx context.Context, attemptID uuid.UUID) (bool, error) {
	exec := db.ExecutorFromContext(ctx, r.db)
	var exists bool
	err := exec.QueryRowContext(ctx,
		`SELECT EXISTS (
		    SELECT 1 FROM answer_evaluation ae
		    JOIN answer_submission s ON s.id = ae.submission_id
		    WHERE s.attempt_id = $1 AND ae.source = 'llm' AND ae.status = 'pending'
		)`,
		attemptID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("has pending llm evaluations: %w", err)
	}
	return exists, nil
}
