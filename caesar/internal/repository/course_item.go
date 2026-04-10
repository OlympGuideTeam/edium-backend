package repository

import (
	"context"
	"database/sql"
	"errors"

	"caesar/internal/domain"
	"caesar/internal/infra/db"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

func (s *PgCourseStore) CreateItem(ctx context.Context, moduleID, refID uuid.UUID, t domain.CourseItemType, orderIndex int) (uuid.UUID, error) {
	exec := db.ExecutorFromContext(ctx, s.db)
	var id uuid.UUID
	err := exec.QueryRowContext(ctx,
		`INSERT INTO course_item (module_id, ref_id, type, order_index) VALUES ($1, $2, $3, $4) RETURNING id`,
		moduleID, refID, t, orderIndex,
	).Scan(&id)
	return id, err
}

func (s *PgCourseStore) GetItemByID(ctx context.Context, id uuid.UUID) (*domain.CourseItem, error) {
	exec := db.ExecutorFromContext(ctx, s.db)
	var item domain.CourseItem
	err := exec.QueryRowContext(ctx,
		`SELECT id, module_id, ref_id, type, order_index, settings FROM course_item WHERE id = $1`,
		id,
	).Scan(&item.ID, &item.ModuleID, &item.RefID, &item.Type, &item.OrderIndex, &item.Settings)
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
		SELECT ci.id, ci.ref_id
		FROM course_item ci
		JOIN course_module cm ON cm.id = ci.module_id
		WHERE cm.course_id = $1
		ORDER BY cm.created_at, ci.order_index
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
		SELECT ci.id, ci.module_id, ci.ref_id, ci.type, ci.order_index,
		       p.attempt_id, p.score
		FROM course_item ci
		LEFT JOIN course_user_item_progress p
		       ON p.course_item_id = ci.id AND p.user_id = $2
		WHERE ci.module_id = ANY($1::uuid[])
		ORDER BY ci.module_id, ci.order_index
	`, pq.Array(ids), userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var items []domain.CourseModuleItem
	for rows.Next() {
		var item domain.CourseModuleItem
		if err := rows.Scan(
			&item.ID, &item.ModuleID, &item.RefID, &item.Type, &item.OrderIndex,
			&item.AttemptID, &item.Score,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
