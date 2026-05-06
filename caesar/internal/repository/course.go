package repository

import (
	"context"
	"database/sql"
	"errors"

	"caesar/internal/domain"
	"caesar/internal/infra/db"

	"github.com/google/uuid"
)

type PgCourseStore struct {
	db *sql.DB
}

func NewPgCourseStore(database *sql.DB) *PgCourseStore {
	return &PgCourseStore{db: database}
}

func (s *PgCourseStore) Create(ctx context.Context, classID, ownerID uuid.UUID, title string) (uuid.UUID, error) {
	exec := db.ExecutorFromContext(ctx, s.db)
	var id uuid.UUID
	err := exec.QueryRowContext(ctx,
		`INSERT INTO course (class_id, owner_id, title) VALUES ($1, $2, $3) RETURNING id`,
		classID, ownerID, title,
	).Scan(&id)
	return id, err
}

func (s *PgCourseStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Course, error) {
	exec := db.ExecutorFromContext(ctx, s.db)
	var c domain.Course
	err := exec.QueryRowContext(ctx,
		`SELECT id, class_id, owner_id, title, module_count, element_count FROM course WHERE id = $1`,
		id,
	).Scan(&c.ID, &c.ClassID, &c.OwnerID, &c.Title, &c.ModuleCount, &c.ElementCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &c, err
}

func (s *PgCourseStore) GetDetail(ctx context.Context, id uuid.UUID) (*domain.CourseDetail, error) {
	exec := db.ExecutorFromContext(ctx, s.db)
	var d domain.CourseDetail
	err := exec.QueryRowContext(ctx, `
		SELECT c.id, c.class_id, c.owner_id, c.title, c.module_count, c.element_count,
		       u.name || ' ' || u.surname
		FROM course c
		JOIN "user" u ON u.id = c.owner_id
		WHERE c.id = $1
	`, id).Scan(
		&d.ID, &d.ClassID, &d.OwnerID, &d.Title, &d.ModuleCount, &d.ElementCount,
		&d.TeacherName,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &d, err
}

func (s *PgCourseStore) ListByClassID(ctx context.Context, classID uuid.UUID) ([]domain.CourseListItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.class_id, c.owner_id, c.title, c.module_count, c.element_count,
		       u.name || ' ' || u.surname
		FROM course c
		JOIN "user" u ON u.id = c.owner_id
		WHERE c.class_id = $1
		ORDER BY c.created_at
	`, classID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanCourseListItems(rows)
}

func (s *PgCourseStore) ListByStudentID(ctx context.Context, userID uuid.UUID) ([]domain.CourseListItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.class_id, c.owner_id, c.title, c.module_count, c.element_count,
		       u.name || ' ' || u.surname
		FROM course c
		JOIN "user" u ON u.id = c.owner_id
		JOIN class_member cm ON cm.class_id = c.class_id
		WHERE cm.user_id = $1 AND cm.role = 'student'
		ORDER BY c.created_at
	`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanCourseListItems(rows)
}

func (s *PgCourseStore) Update(ctx context.Context, id uuid.UUID, title string) error {
	exec := db.ExecutorFromContext(ctx, s.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE course SET title = $1 WHERE id = $2`,
		title, id,
	)
	return err
}

func (s *PgCourseStore) Delete(ctx context.Context, id uuid.UUID) error {
	exec := db.ExecutorFromContext(ctx, s.db)
	_, err := exec.ExecContext(ctx, `DELETE FROM course WHERE id = $1`, id)
	return err
}

func (s *PgCourseStore) GetStudentIDsByCourseID(ctx context.Context, courseID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT cm.user_id
		 FROM class_member cm
		 JOIN course c ON c.class_id = cm.class_id
		 WHERE c.id = $1 AND cm.role = 'student'`,
		courseID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func scanCourseListItems(rows *sql.Rows) ([]domain.CourseListItem, error) {
	var items []domain.CourseListItem
	for rows.Next() {
		var item domain.CourseListItem
		if err := rows.Scan(
			&item.ID, &item.ClassID, &item.OwnerID, &item.Title,
			&item.ModuleCount, &item.ElementCount, &item.TeacherName,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
