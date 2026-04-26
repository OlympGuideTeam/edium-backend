package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"louvre/internal/domain"
)

type ImageRepository interface {
	Save(ctx context.Context, img domain.Image) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Image, error)
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

type PgImageRepository struct {
	db *sql.DB
}

func NewPgImageRepository(db *sql.DB) ImageRepository {
	return &PgImageRepository{db: db}
}

const saveQuery = `
INSERT INTO image (id, file_name, mime_type, size_bytes, width, height, s3_key, bucket_name)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING created_at`

const findByIDQuery = `
SELECT id, file_name, mime_type, size_bytes, width, height, s3_key, bucket_name, created_at, deleted_at
FROM image
WHERE id = $1 AND deleted_at IS NULL`

const countByUserIDQuery = `
SELECT COUNT(*) FROM image WHERE created_at >= NOW() - INTERVAL '1 hour'`

const softDeleteQuery = `
UPDATE image
SET deleted_at = $1
WHERE id = $2 AND deleted_at IS NULL`

func (r *PgImageRepository) Save(ctx context.Context, img domain.Image) error {
	var createdAt time.Time
	err := r.db.QueryRowContext(ctx, saveQuery,
		img.ID, img.FileName, img.MimeType, img.SizeBytes,
		img.Width, img.Height, img.S3Key, img.BucketName,
	).Scan(&createdAt)
	if err != nil {
		return fmt.Errorf("сохранение изображения: %w", err)
	}
	img.CreatedAt = createdAt
	return nil
}

func (r *PgImageRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Image, error) {
	var img domain.Image
	var deletedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, findByIDQuery, id).Scan(
		&img.ID, &img.FileName, &img.MimeType, &img.SizeBytes,
		&img.Width, &img.Height, &img.S3Key, &img.BucketName,
		&img.CreatedAt, &deletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("поиск изображения по ID: %w", err)
	}

	if deletedAt.Valid {
		img.DeletedAt = &deletedAt.Time
	}

	return &img, nil
}

func (r *PgImageRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, countByUserIDQuery).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("подсчёт загрузок пользователя: %w", err)
	}
	return count, nil
}

func (r *PgImageRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, softDeleteQuery, time.Now(), id)
	if err != nil {
		return fmt.Errorf("мягкое удаление изображения: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
