package repository

import (
	"caesar/internal/domain"
	"caesar/internal/infra/db"
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

type PgUserStore struct {
	db *sql.DB
}

func NewPgUserStore(database *sql.DB) *PgUserStore {
	return &PgUserStore{db: database}
}

func (s *PgUserStore) Create(ctx context.Context, u domain.User) error {
	exec := db.ExecutorFromContext(ctx, s.db)
	_, err := exec.ExecContext(ctx, `
		INSERT INTO "user" (id, name, surname, phone, status)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING
	`, u.ID, u.Name, u.Surname, u.Phone, u.Status)
	return err
}

func (s *PgUserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	exec := db.ExecutorFromContext(ctx, s.db)
	row := exec.QueryRowContext(ctx,
		`SELECT id, name, surname, phone, status FROM "user" WHERE id = $1`,
		id,
	)

	var u domain.User
	if err := row.Scan(&u.ID, &u.Name, &u.Surname, &u.Phone, &u.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (s *PgUserStore) Update(ctx context.Context, id uuid.UUID, name, surname string) error {
	exec := db.ExecutorFromContext(ctx, s.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE "user" SET name = $1, surname = $2 WHERE id = $3`,
		name, surname, id,
	)
	return err
}

func (s *PgUserStore) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.UserStatus) error {
	exec := db.ExecutorFromContext(ctx, s.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE "user" SET status = $1 WHERE id = $2`,
		status, id,
	)
	return err
}

func (s *PgUserStore) GetStatistic(ctx context.Context, id uuid.UUID) (*domain.UserStatistic, error) {
	var st domain.UserStatistic
	err := s.db.QueryRowContext(ctx, `
		SELECT
		    (SELECT COUNT(*) FROM class WHERE owner_id = $1),
		    (SELECT COALESCE(SUM(student_count), 0) FROM class WHERE owner_id = $1),
		    (SELECT COUNT(*) FROM course WHERE owner_id = $1),
		    (SELECT COUNT(DISTINCT c.id) FROM course c
		     JOIN class_member cm ON cm.class_id = c.class_id
		     WHERE cm.user_id = $1 AND cm.role = 'student')
	`, id).Scan(
		&st.ClassTeacherCount,
		&st.StudentCount,
		&st.CourseTeacherCount,
		&st.CourseStudentCount,
	)
	if err != nil {
		return nil, err
	}
	return &st, nil
}
