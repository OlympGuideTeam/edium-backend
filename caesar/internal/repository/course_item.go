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

func (s *PgCourseStore) CreateItem(ctx context.Context, moduleID, objectID uuid.UUID, t domain.CourseItemType) (uuid.UUID, error) {
	exec := db.ExecutorFromContext(ctx, s.db)
	var id uuid.UUID
	err := exec.QueryRowContext(ctx,
		`INSERT INTO course_item (module_id, object_id, type) VALUES ($1, $2, $3) RETURNING id`,
		moduleID, objectID, t,
	).Scan(&id)
	return id, err
}

func (s *PgCourseStore) GetItemByID(ctx context.Context, id uuid.UUID) (*domain.CourseItem, error) {
	exec := db.ExecutorFromContext(ctx, s.db)
	var item domain.CourseItem
	err := exec.QueryRowContext(ctx,
		`SELECT id, module_id, object_id, type, settings FROM course_item WHERE id = $1`,
		id,
	).Scan(&item.ID, &item.ModuleID, &item.ObjectID, &item.Type, &item.Settings)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &item, err
}

func (s *PgCourseStore) FindItemByObjectID(ctx context.Context, objectID uuid.UUID) (*domain.CourseItem, error) {
	exec := db.ExecutorFromContext(ctx, s.db)
	var item domain.CourseItem
	err := exec.QueryRowContext(ctx,
		`SELECT id, module_id, object_id, type, settings FROM course_item WHERE object_id = $1 LIMIT 1`,
		objectID,
	).Scan(&item.ID, &item.ModuleID, &item.ObjectID, &item.Type, &item.Settings)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &item, err
}

func (s *PgCourseStore) DeleteItem(ctx context.Context, id uuid.UUID) error {
	exec := db.ExecutorFromContext(ctx, s.db)
	_, err := exec.ExecContext(ctx, `DELETE FROM course_item WHERE id = $1`, id)
	return err
}

func (s *PgCourseStore) GetSheetItems(ctx context.Context, courseID uuid.UUID) ([]domain.CourseSheetItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ci.id, ci.object_id
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
		if err := rows.Scan(&item.ID, &item.RefID); err != nil {
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
		       p.attempt_id, p.score
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
		if err := rows.Scan(
			&item.ID, &item.ModuleID, &item.ObjectID, &item.Type,
			&item.AttemptID, &item.Score,
		); err != nil {
			return nil, err
		}
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

func (s *PgCourseStore) UpdateItemSettings(ctx context.Context, id uuid.UUID, settings json.RawMessage) error {
	exec := db.ExecutorFromContext(ctx, s.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE course_item SET settings = $1 WHERE id = $2`,
		settings, id,
	)
	return err
}
