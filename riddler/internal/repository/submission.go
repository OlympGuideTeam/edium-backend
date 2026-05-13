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

func (r *PgAttemptRepository) GetSubmissionByID(ctx context.Context, id uuid.UUID) (*domain.AnswerSubmission, error) {
	exec := db.ExecutorFromContext(ctx, r.db)
	var s domain.AnswerSubmission
	var dataJSON []byte
	var finalSource sql.NullString
	var finalFeedback sql.NullString
	err := exec.QueryRowContext(ctx,
		`SELECT id, attempt_id, question_id, answer_data, final_score, final_source, final_feedback
		 FROM answer_submission WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.AttemptID, &s.QuestionID, &dataJSON, &s.FinalScore, &finalSource, &finalFeedback)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get submission: %w", err)
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
	return &s, nil
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

func (r *PgAttemptRepository) UpdateSubmissionFinalScore(ctx context.Context, submissionID uuid.UUID, score float64, source domain.FinalSource, feedback *string) error {
	exec := db.ExecutorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE answer_submission
		 SET final_score = $2, final_source = $3, final_feedback = $4, updated_at = now()
		 WHERE id = $1
		   AND (final_source IS NULL OR final_source != 'teacher')`,
		submissionID, score, source, feedback,
	)
	if err != nil {
		return fmt.Errorf("update submission final score: %w", err)
	}
	return nil
}

func (r *PgAttemptRepository) SumScores(ctx context.Context, attemptID uuid.UUID) (float64, error) {
	exec := db.ExecutorFromContext(ctx, r.db)
	var total float64
	err := exec.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(final_score), 0) FROM answer_submission WHERE attempt_id = $1`,
		attemptID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum scores: %w", err)
	}
	return total, nil
}

func (r *PgAttemptRepository) GetAnswersWithQuestion(ctx context.Context, attemptID, quizTemplateID uuid.UUID) ([]domain.AnswerWithQuestion, error) {
	exec := db.ExecutorFromContext(ctx, r.db)
	rows, err := exec.QueryContext(ctx,
		`SELECT q.id, q.type, q.text,
		        s.id, s.answer_data, s.final_score, s.final_source, s.final_feedback
		 FROM question q
		 LEFT JOIN answer_submission s ON s.question_id = q.id AND s.attempt_id = $1
		 WHERE q.quiz_template_id = $2
		 ORDER BY q.order_index`,
		attemptID, quizTemplateID,
	)
	if err != nil {
		return nil, fmt.Errorf("get answers with question: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.AnswerWithQuestion
	for rows.Next() {
		var a domain.AnswerWithQuestion
		var sidStr sql.NullString
		var dataJSON []byte
		var finalScore sql.NullFloat64
		var finalSource sql.NullString
		var finalFeedback sql.NullString
		if err := rows.Scan(&a.QuestionID, &a.QuestionType, &a.QuestionText,
			&sidStr, &dataJSON, &finalScore, &finalSource, &finalFeedback); err != nil {
			return nil, fmt.Errorf("scan answer with question: %w", err)
		}
		if sidStr.Valid {
			sid, _ := uuid.Parse(sidStr.String)
			a.SubmissionID = &sid
		}
		if dataJSON != nil {
			if err := json.Unmarshal(dataJSON, &a.AnswerData); err != nil {
				return nil, fmt.Errorf("unmarshal answer_data: %w", err)
			}
		}
		if finalScore.Valid {
			a.FinalScore = &finalScore.Float64
		}
		if finalSource.Valid {
			src := domain.FinalSource(finalSource.String)
			a.FinalSource = &src
		}
		if finalFeedback.Valid {
			a.FinalFeedback = &finalFeedback.String
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
