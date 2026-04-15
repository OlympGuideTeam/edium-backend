package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"caesar/internal/domain"
	"caesar/internal/infra/db"

	"github.com/google/uuid"
)

func (s *PgCourseStore) CreateModule(ctx context.Context, courseID uuid.UUID, title string) (uuid.UUID, error) {
	exec := db.ExecutorFromContext(ctx, s.db)
	var id uuid.UUID
	err := exec.QueryRowContext(ctx, `
		INSERT INTO course_module (course_id, title, order_index)
		VALUES ($1, $2, (SELECT COALESCE(MAX(order_index), 0) + 1 FROM course_module WHERE course_id = $1))
		RETURNING id
	`, courseID, title).Scan(&id)
	return id, err
}

func (s *PgCourseStore) GetModuleByID(ctx context.Context, id uuid.UUID) (*domain.CourseModule, error) {
	exec := db.ExecutorFromContext(ctx, s.db)
	var m domain.CourseModule
	err := exec.QueryRowContext(ctx,
		`SELECT id, course_id, title, element_count, order_index FROM course_module WHERE id = $1`,
		id,
	).Scan(&m.ID, &m.CourseID, &m.Title, &m.ElementCount, &m.OrderIndex)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &m, err
}

func (s *PgCourseStore) ListModules(ctx context.Context, courseID uuid.UUID) ([]domain.CourseModule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, course_id, title, element_count, order_index
		FROM course_module
		WHERE course_id = $1
		ORDER BY order_index
	`, courseID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var modules []domain.CourseModule
	for rows.Next() {
		var m domain.CourseModule
		if err := rows.Scan(&m.ID, &m.CourseID, &m.Title, &m.ElementCount, &m.OrderIndex); err != nil {
			return nil, err
		}
		modules = append(modules, m)
	}
	return modules, rows.Err()
}

func (s *PgCourseStore) UpdateModule(ctx context.Context, id uuid.UUID, title string) error {
	exec := db.ExecutorFromContext(ctx, s.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE course_module SET title = $1 WHERE id = $2`,
		title, id,
	)
	return err
}

func (s *PgCourseStore) DeleteModule(ctx context.Context, id uuid.UUID) error {
	exec := db.ExecutorFromContext(ctx, s.db)
	_, err := exec.ExecContext(ctx, `DELETE FROM course_module WHERE id = $1`, id)
	return err
}

func (s *PgCourseStore) ReorderModules(ctx context.Context, courseID uuid.UUID, moduleIDs []uuid.UUID) error {
	ids := make([]string, len(moduleIDs))
	for i, id := range moduleIDs {
		ids[i] = id.String()
	}
	exec := db.ExecutorFromContext(ctx, s.db)
	_, err := exec.ExecContext(ctx, `
		UPDATE course_module
		SET order_index = idx.pos
		FROM (
			SELECT u.id, u.pos
			FROM unnest($1::uuid[]) WITH ORDINALITY AS u(id, pos)
		) AS idx
		WHERE course_module.id = idx.id AND course_module.course_id = $2
	`, "{"+strings.Join(ids, ",")+"}", courseID)
	return err
}
