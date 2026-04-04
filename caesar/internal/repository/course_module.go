package repository

import (
	"context"
	"database/sql"
	"errors"

	"caesar/internal/domain"
	"caesar/internal/infra/db"

	"github.com/google/uuid"
)

func (s *PgCourseStore) CreateModule(ctx context.Context, courseID uuid.UUID, title string) (uuid.UUID, error) {
	exec := db.ExecutorFromContext(ctx, s.db)
	var id uuid.UUID
	err := exec.QueryRowContext(ctx,
		`INSERT INTO course_module (course_id, title) VALUES ($1, $2) RETURNING id`,
		courseID, title,
	).Scan(&id)
	return id, err
}

func (s *PgCourseStore) GetModuleByID(ctx context.Context, id uuid.UUID) (*domain.CourseModule, error) {
	exec := db.ExecutorFromContext(ctx, s.db)
	var m domain.CourseModule
	err := exec.QueryRowContext(ctx,
		`SELECT id, course_id, title, element_count FROM course_module WHERE id = $1`,
		id,
	).Scan(&m.ID, &m.CourseID, &m.Title, &m.ElementCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &m, err
}

func (s *PgCourseStore) ListModules(ctx context.Context, courseID uuid.UUID) ([]domain.CourseModule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, course_id, title, element_count
		FROM course_module
		WHERE course_id = $1
		ORDER BY created_at
	`, courseID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var modules []domain.CourseModule
	for rows.Next() {
		var m domain.CourseModule
		if err := rows.Scan(&m.ID, &m.CourseID, &m.Title, &m.ElementCount); err != nil {
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
