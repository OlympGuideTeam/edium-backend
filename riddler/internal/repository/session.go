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
		`SELECT id, quiz_template_id, mode, source, status, max_score, total_time_limit_sec, question_time_limit_sec,
		        shuffle_questions, started_at, finished_at
		 FROM quiz_session WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.QuizTemplateID, &s.Mode, &s.Source, &s.Status, &s.MaxScore,
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
		 (quiz_template_id, mode, source, max_score, total_time_limit_sec, question_time_limit_sec,
		  shuffle_questions, status, settings, started_at, finished_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, COALESCE($10, now()), $11)
		 RETURNING id`,
		p.QuizTemplateID,
		p.Mode,
		p.Source,
		p.MaxScore,
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

func (r *PgSessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	exec := db.ExecutorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx, `DELETE FROM quiz_session WHERE id = $1`, id)
	return err
}

func (r *PgSessionRepository) HasAttempts(ctx context.Context, sessionID uuid.UUID) (bool, error) {
	exec := db.ExecutorFromContext(ctx, r.db)
	var exists bool
	err := exec.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM attempt WHERE session_id = $1)`,
		sessionID,
	).Scan(&exists)
	return exists, err
}

func (r *PgSessionRepository) HasSessions(ctx context.Context, quizTemplateID uuid.UUID) (bool, error) {
	exec := db.ExecutorFromContext(ctx, r.db)
	var exists bool
	err := exec.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM quiz_session WHERE quiz_template_id = $1)`,
		quizTemplateID,
	).Scan(&exists)
	return exists, err
}

func (r *PgSessionRepository) FindFinishedNeedingGrading(ctx context.Context) ([]domain.QuizSession, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT qs.id, qs.quiz_template_id, qs.mode, qs.status,
		        qs.total_time_limit_sec, qs.question_time_limit_sec,
		        qs.shuffle_questions, qs.started_at, qs.finished_at
		 FROM quiz_session qs
		 JOIN quiz_template qt ON qt.id = qs.quiz_template_id
		 WHERE qt.need_evaluation = true
		   AND qs.grading_sent_at IS NULL
		   AND qs.status = 'finished'`,
	)
	if err != nil {
		return nil, fmt.Errorf("find finished needing grading: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.QuizSession
	for rows.Next() {
		var s domain.QuizSession
		if err := rows.Scan(&s.ID, &s.QuizTemplateID, &s.Mode, &s.Status,
			&s.TotalTimeLimitSec, &s.QuestionTimeLimitSec,
			&s.ShuffleQuestions, &s.StartedAt, &s.FinishedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *PgSessionRepository) SetGradingSent(ctx context.Context, id uuid.UUID) error {
	exec := db.ExecutorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE quiz_session SET grading_sent_at = now() WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("set grading sent: %w", err)
	}
	return nil
}

func (r *PgSessionRepository) GetFreeAnswerSubmissionsForSession(ctx context.Context, sessionID uuid.UUID) ([]domain.FreeAnswerSubmission, error) {
	exec := db.ExecutorFromContext(ctx, r.db)
	rows, err := exec.QueryContext(ctx,
		`SELECT s.id, s.attempt_id, q.id, q.text, s.answer_data->>'text'
		 FROM answer_submission s
		 JOIN attempt a ON a.id = s.attempt_id
		 JOIN question q ON q.id = s.question_id
		 WHERE a.session_id = $1
		   AND a.status = 'grading'
		   AND q.type = 'with_free_answer'`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("get free answer submissions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.FreeAnswerSubmission
	for rows.Next() {
		var f domain.FreeAnswerSubmission
		if err := rows.Scan(&f.SubmissionID, &f.AttemptID, &f.QuestionID, &f.QuestionText, &f.AnswerText); err != nil {
			return nil, fmt.Errorf("scan free answer submission: %w", err)
		}
		result = append(result, f)
	}
	return result, rows.Err()
}
