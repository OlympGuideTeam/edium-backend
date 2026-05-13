package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

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
		`SELECT id, quiz_template_id, mode, source, teacher_id, status, max_score, total_time_limit_sec, question_time_limit_sec,
		        shuffle_questions, started_at, finished_at
		 FROM quiz_session WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.QuizTemplateID, &s.Mode, &s.Source, &s.TeacherID, &s.Status, &s.MaxScore,
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
		 (quiz_template_id, mode, source, teacher_id, max_score, total_time_limit_sec, question_time_limit_sec,
		  shuffle_questions, status, settings, started_at, finished_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, COALESCE($11, now()), $12)
		 RETURNING id`,
		p.QuizTemplateID,
		p.Mode,
		p.Source,
		p.TeacherID,
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

func (r *PgSessionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.SessionStatus) error {
	exec := db.ExecutorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE quiz_session SET status = $2, updated_at = now() WHERE id = $1`,
		id, status,
	)
	if err != nil {
		return fmt.Errorf("update session status: %w", err)
	}
	return nil
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

func (r *PgSessionRepository) GetManyByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.QuizSession, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = id.String()
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, mode, status, started_at, finished_at
		 FROM quiz_session WHERE id = ANY($1::uuid[])`,
		"{"+strings.Join(strs, ",")+"}")
	if err != nil {
		return nil, fmt.Errorf("get sessions by ids: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.QuizSession
	for rows.Next() {
		var s domain.QuizSession
		if err := rows.Scan(&s.ID, &s.Mode, &s.Status, &s.StartedAt, &s.FinishedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
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

func (r *PgSessionRepository) FindSessionsReadyToAutoClose(ctx context.Context) ([]domain.QuizSession, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT qs.id, qs.quiz_template_id, qs.mode, qs.source, qs.teacher_id, qs.status, qs.max_score,
		        qs.total_time_limit_sec, qs.question_time_limit_sec, qs.shuffle_questions, qs.started_at, qs.finished_at
		 FROM quiz_session qs
		 WHERE qs.mode = 'test'
		   AND qs.source = 'course'
		   AND qs.status = 'active'
		   AND qs.finished_at IS NOT NULL
		   AND qs.finished_at < now()
		   AND NOT EXISTS (
		     SELECT 1 FROM attempt a
		     WHERE a.session_id = qs.id AND a.status = 'in_progress'
		   )`,
	)
	if err != nil {
		return nil, fmt.Errorf("find sessions ready to auto close: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.QuizSession
	for rows.Next() {
		var s domain.QuizSession
		if err := rows.Scan(&s.ID, &s.QuizTemplateID, &s.Mode, &s.Source, &s.TeacherID, &s.Status, &s.MaxScore,
			&s.TotalTimeLimitSec, &s.QuestionTimeLimitSec, &s.ShuffleQuestions, &s.StartedAt, &s.FinishedAt); err != nil {
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

func (r *PgSessionRepository) ListLiveSessions(ctx context.Context, authorID uuid.UUID, source *string, limit int) ([]domain.LiveSession, error) {
	exec := db.ExecutorFromContext(ctx, r.db)
	var sourceArg any
	if source != nil {
		sourceArg = *source
	}
	rows, err := exec.QueryContext(ctx,
		`SELECT qs.id, qs.quiz_template_id, qt.title, qs.source, qs.status, qs.created_at
		 FROM quiz_session qs
		 JOIN quiz_template qt ON qt.id = qs.quiz_template_id
		 WHERE qs.mode = 'live'
		   AND qs.teacher_id = $1
		   AND ($2::text IS NULL OR qs.source::text = $2)
		 ORDER BY qs.created_at DESC
		 LIMIT $3`,
		authorID, sourceArg, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list live sessions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.LiveSession
	for rows.Next() {
		var s domain.LiveSession
		if err := rows.Scan(&s.SessionID, &s.QuizTemplateID, &s.QuizTitle, &s.Source, &s.Status, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan session row: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
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

func (r *PgSessionRepository) FindRunningCourseLiveSessions(ctx context.Context, courseIDs []uuid.UUID) ([]domain.LiveLobbySnapshot, error) {
	if len(courseIDs) == 0 {
		return nil, nil
	}
	strs := make([]string, len(courseIDs))
	for i, id := range courseIDs {
		strs[i] = id.String()
	}
	exec := db.ExecutorFromContext(ctx, r.db)
	rows, err := exec.QueryContext(ctx,
		`SELECT qs.id, qt.course_id, qt.title, qs.question_time_limit_sec
		 FROM quiz_session qs
		 JOIN quiz_template qt ON qt.id = qs.quiz_template_id
		 WHERE qs.mode = 'live'
		   AND qs.source = 'course'
		   AND qs.status = 'running'
		   AND qt.course_id = ANY($1::uuid[])`,
		"{"+strings.Join(strs, ",")+"}")
	if err != nil {
		return nil, fmt.Errorf("find running course live sessions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.LiveLobbySnapshot
	for rows.Next() {
		var s domain.LiveLobbySnapshot
		var limitSec *int
		if err := rows.Scan(&s.SessionID, &s.CourseID, &s.QuizTitle, &limitSec); err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}
		if limitSec != nil {
			s.QuestionTimeLimitSec = *limitSec
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *PgSessionRepository) FindStudentRecentGrades(ctx context.Context, userID uuid.UUID, limit int) ([]domain.RecentGradeItem, error) {
	exec := db.ExecutorFromContext(ctx, r.db)
	rows, err := exec.QueryContext(ctx,
		`SELECT qs.id, qs.quiz_template_id, qt.title,
		        a.id, a.grade, a.status, a.finished_at
		 FROM attempt a
		 JOIN quiz_session qs ON qs.id = a.session_id
		 JOIN quiz_template qt ON qt.id = qs.quiz_template_id
		 WHERE a.user_id = $1
		   AND qs.source = $2
		   AND qs.mode = $3
		   AND a.status != $4
		 ORDER BY a.finished_at DESC NULLS LAST
		 LIMIT $5`,
		userID, domain.LiveSourceCourse, domain.SessionModeTest, domain.AttemptStatusInProgress, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("find student recent grades: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.RecentGradeItem
	for rows.Next() {
		var item domain.RecentGradeItem
		if err := rows.Scan(&item.SessionID, &item.QuizTemplateID, &item.QuizTitle,
			&item.AttemptID, &item.Score, &item.Status, &item.FinishedAt); err != nil {
			return nil, fmt.Errorf("scan recent grade: %w", err)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *PgSessionRepository) FindStudentActiveTests(ctx context.Context, userID uuid.UUID) ([]domain.ActiveTestItem, error) {
	exec := db.ExecutorFromContext(ctx, r.db)
	rows, err := exec.QueryContext(ctx,
		`SELECT qs.id, qs.quiz_template_id, qt.title,
		        qs.total_time_limit_sec, qs.started_at, qs.finished_at,
		        a.id, a.status
		 FROM quiz_session qs
		 JOIN quiz_template qt ON qt.id = qs.quiz_template_id
		 JOIN attempt a ON a.session_id = qs.id AND a.user_id = $1
		 WHERE qs.source = $2
		   AND qs.mode = $3
		   AND qs.status = $4
		   AND (qs.finished_at IS NULL OR qs.finished_at > now())
		   AND a.status = $5
		 ORDER BY qs.started_at DESC
		 LIMIT 5`,
		userID, domain.LiveSourceCourse, domain.SessionModeTest,
		domain.SessionStatusActive, domain.AttemptStatusInProgress,
	)
	if err != nil {
		return nil, fmt.Errorf("find student active tests: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.ActiveTestItem
	for rows.Next() {
		var item domain.ActiveTestItem
		var attemptID *uuid.UUID
		var attemptStatus *string
		if err := rows.Scan(&item.SessionID, &item.QuizTemplateID, &item.QuizTitle,
			&item.TotalTimeLimitSec, &item.SessionStartedAt, &item.SessionFinishedAt,
			&attemptID, &attemptStatus); err != nil {
			return nil, fmt.Errorf("scan active test: %w", err)
		}
		item.AttemptID = attemptID
		if attemptStatus != nil {
			s := domain.AttemptStatus(*attemptStatus)
			item.AttemptStatus = &s
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *PgSessionRepository) FindAwaitingReview(ctx context.Context, authorID uuid.UUID) ([]domain.AwaitingReviewSession, error) {
	exec := db.ExecutorFromContext(ctx, r.db)
	rows, err := exec.QueryContext(ctx,
		`SELECT qs.id, qs.quiz_template_id, qt.title,
		        COUNT(*) FILTER (WHERE a.status = 'grading')   AS grading_count,
		        COUNT(*) FILTER (WHERE a.status = 'graded')    AS graded_count,
		        COUNT(*) FILTER (WHERE a.status = 'completed') AS completed_count
		 FROM quiz_session qs
		 JOIN quiz_template qt ON qt.id = qs.quiz_template_id
		 JOIN attempt a ON a.session_id = qs.id
		 WHERE qt.author_id = $1
		   AND a.status IN ('grading', 'graded', 'completed')
		 GROUP BY qs.id, qs.quiz_template_id, qt.title
		 HAVING COUNT(*) FILTER (WHERE a.status IN ('grading', 'graded')) > 0
		 ORDER BY qs.created_at DESC`,
		authorID,
	)
	if err != nil {
		return nil, fmt.Errorf("find awaiting review sessions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.AwaitingReviewSession
	for rows.Next() {
		var s domain.AwaitingReviewSession
		if err := rows.Scan(&s.SessionID, &s.QuizTemplateID, &s.QuizTitle, &s.GradingCount, &s.GradedCount, &s.CompletedCount); err != nil {
			return nil, fmt.Errorf("scan awaiting review session: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}
