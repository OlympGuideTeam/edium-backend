package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"caesar/internal/domain"
	"caesar/internal/infra/db"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

func (s *PgCourseStore) CreateItem(ctx context.Context, moduleID, objectID uuid.UUID, t domain.CourseItemType, payload json.RawMessage) (uuid.UUID, error) {
	exec := db.ExecutorFromContext(ctx, s.db)
	var id uuid.UUID
	err := exec.QueryRowContext(ctx,
		`INSERT INTO course_item (module_id, object_id, type, payload) VALUES ($1, $2, $3, $4) RETURNING id`,
		moduleID, objectID, t, []byte(payload),
	).Scan(&id)
	return id, err
}

func (s *PgCourseStore) GetItemByID(ctx context.Context, id uuid.UUID) (*domain.CourseItem, error) {
	exec := db.ExecutorFromContext(ctx, s.db)
	var item domain.CourseItem
	err := exec.QueryRowContext(ctx,
		`SELECT id, module_id, object_id, type FROM course_item WHERE id = $1`,
		id,
	).Scan(&item.ID, &item.ModuleID, &item.ObjectID, &item.Type)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &item, err
}

func (s *PgCourseStore) FindItemByObjectID(ctx context.Context, objectID uuid.UUID) (*domain.CourseItem, error) {
	exec := db.ExecutorFromContext(ctx, s.db)
	var item domain.CourseItem
	err := exec.QueryRowContext(ctx,
		`SELECT id, module_id, object_id, type FROM course_item WHERE object_id = $1 LIMIT 1`,
		objectID,
	).Scan(&item.ID, &item.ModuleID, &item.ObjectID, &item.Type)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &item, err
}

func (s *PgCourseStore) UpsertCourseDraft(ctx context.Context, quizTemplateID, courseID uuid.UUID, title string, payload json.RawMessage) (uuid.UUID, error) {
	exec := db.ExecutorFromContext(ctx, s.db)
	var id uuid.UUID
	var payloadArg interface{}
	if len(payload) > 0 {
		payloadArg = []byte(payload)
	}
	err := exec.QueryRowContext(ctx,
		`INSERT INTO course_draft (quiz_template_id, course_id, title, payload) VALUES ($1, $2, $3, $4)
		 ON CONFLICT (quiz_template_id, course_id) DO UPDATE
		     SET title   = EXCLUDED.title,
		         payload = EXCLUDED.payload
		 RETURNING id`,
		quizTemplateID, courseID, title, payloadArg,
	).Scan(&id)
	return id, err
}

func (s *PgCourseStore) GetDraftByID(ctx context.Context, id uuid.UUID) (*domain.CourseDraft, error) {
	exec := db.ExecutorFromContext(ctx, s.db)
	var d domain.CourseDraft
	var payload []byte
	err := exec.QueryRowContext(ctx,
		`SELECT id, quiz_template_id, course_id, title, payload FROM course_draft WHERE id = $1`,
		id,
	).Scan(&d.ID, &d.QuizTemplateID, &d.CourseID, &d.Title, &payload)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d.Payload = json.RawMessage(payload)
	return &d, nil
}

func (s *PgCourseStore) DeleteDraft(ctx context.Context, id uuid.UUID) error {
	exec := db.ExecutorFromContext(ctx, s.db)
	_, err := exec.ExecContext(ctx, `DELETE FROM course_draft WHERE id = $1`, id)
	return err
}

func (s *PgCourseStore) DeleteDraftByTemplateAndCourse(ctx context.Context, quizTemplateID, courseID uuid.UUID) error {
	exec := db.ExecutorFromContext(ctx, s.db)
	_, err := exec.ExecContext(ctx,
		`DELETE FROM course_draft WHERE quiz_template_id = $1 AND course_id = $2`,
		quizTemplateID, courseID,
	)
	return err
}

func (s *PgCourseStore) ListDraftsByCourseID(ctx context.Context, courseID uuid.UUID) ([]domain.CourseDraft, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, quiz_template_id, course_id, title, payload FROM course_draft WHERE course_id = $1 ORDER BY created_at`,
		courseID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var drafts []domain.CourseDraft
	for rows.Next() {
		var d domain.CourseDraft
		var payload []byte
		if err := rows.Scan(&d.ID, &d.QuizTemplateID, &d.CourseID, &d.Title, &payload); err != nil {
			return nil, err
		}
		d.Payload = json.RawMessage(payload)
		drafts = append(drafts, d)
	}
	return drafts, rows.Err()
}

func (s *PgCourseStore) DeleteItem(ctx context.Context, id uuid.UUID) error {
	exec := db.ExecutorFromContext(ctx, s.db)
	_, err := exec.ExecContext(ctx, `DELETE FROM course_item WHERE id = $1`, id)
	return err
}

func (s *PgCourseStore) GetSheetItems(ctx context.Context, courseID uuid.UUID) ([]domain.CourseSheetItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ci.id, ci.object_id, ci.payload->>'title'
		FROM course_item ci
		JOIN course_module cm ON cm.id = ci.module_id
		WHERE cm.course_id = $1
		ORDER BY cm.created_at, ci.created_at
	`, courseID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var items []domain.CourseSheetItem
	for rows.Next() {
		var item domain.CourseSheetItem
		if err := rows.Scan(&item.ID, &item.RefID, &item.Title); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PgCourseStore) GetSheetScores(ctx context.Context, courseID uuid.UUID) ([]domain.UserItemScore, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.user_id, p.course_item_id, p.score
		FROM course_user_item_progress p
		JOIN course_item ci ON ci.id = p.course_item_id
		JOIN course_module cm ON cm.id = ci.module_id
		WHERE cm.course_id = $1
	`, courseID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var scores []domain.UserItemScore
	for rows.Next() {
		var s domain.UserItemScore
		if err := rows.Scan(&s.UserID, &s.ItemID, &s.Score); err != nil {
			return nil, err
		}
		scores = append(scores, s)
	}
	return scores, rows.Err()
}

func (s *PgCourseStore) ListItemsByModuleIDs(ctx context.Context, moduleIDs []uuid.UUID, userID uuid.UUID) ([]domain.CourseModuleItem, error) {
	if len(moduleIDs) == 0 {
		return nil, nil
	}

	ids := make([]string, len(moduleIDs))
	for i, id := range moduleIDs {
		ids[i] = id.String()
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT ci.id, ci.module_id, ci.object_id, ci.type,
		       p.attempt_id, p.score, ci.payload
		FROM course_item ci
		LEFT JOIN course_user_item_progress p
		       ON p.course_item_id = ci.id AND p.user_id = $2
		WHERE ci.module_id = ANY($1::uuid[])
		ORDER BY ci.module_id, ci.created_at
	`, pq.Array(ids), userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var items []domain.CourseModuleItem
	for rows.Next() {
		var item domain.CourseModuleItem
		var payload []byte
		if err := rows.Scan(
			&item.ID, &item.ModuleID, &item.ObjectID, &item.Type,
			&item.AttemptID, &item.Score, &payload,
		); err != nil {
			return nil, err
		}
		item.Payload = json.RawMessage(payload)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PgCourseStore) UpsertProgress(ctx context.Context, courseItemID, userID, attemptID uuid.UUID) error {
	exec := db.ExecutorFromContext(ctx, s.db)
	_, err := exec.ExecContext(ctx, `
		INSERT INTO course_user_item_progress (course_item_id, user_id, attempt_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (course_item_id, user_id) DO UPDATE SET attempt_id = EXCLUDED.attempt_id
	`, courseItemID, userID, attemptID)
	return err
}

func (s *PgCourseStore) UpdateProgressScore(ctx context.Context, courseItemID, userID uuid.UUID, score float64) error {
	exec := db.ExecutorFromContext(ctx, s.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE course_user_item_progress SET score = $1 WHERE course_item_id = $2 AND user_id = $3`,
		score, courseItemID, userID,
	)
	return err
}
