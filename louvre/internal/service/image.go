package service

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"mime/multipart"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"

	"louvre/internal/domain"
	"louvre/internal/infra/minio"
	"louvre/internal/repository"
)

type ImageService struct {
	imageRepo      repository.ImageRepository
	minio          *minio.Client
	maxFileSize    int64
	maxWidth       int
	maxHeight      int
	allowedTypes   map[string]bool
	maxUploadsHour int
}

func NewImageService(imageRepo repository.ImageRepository, min *minio.Client, maxSize int64, maxWidth, maxHeight, maxUploadsHour int, allowedTypes []string) *ImageService {
	types := make(map[string]bool)
	for _, t := range allowedTypes {
		types[t] = true
	}

	return &ImageService{
		imageRepo:      imageRepo,
		minio:          min,
		maxFileSize:    maxSize,
		maxWidth:       maxWidth,
		maxHeight:      maxHeight,
		allowedTypes:   types,
		maxUploadsHour: maxUploadsHour,
	}
}

func (s *ImageService) Upload(ctx context.Context, fileHeader *multipart.FileHeader) error {
	if fileHeader.Size > s.maxFileSize {
		return fmt.Errorf("файл слишком большой, максимальный размер %d байт", s.maxFileSize)
	}

	mimeType := fileHeader.Header.Get("Content-Type")
	if !s.allowedTypes[mimeType] {
		return fmt.Errorf("невалидный тип файла: %s", mimeType)
	}

	count, err := s.imageRepo.CountByUserID(ctx, uuid.Nil)
	if err != nil {
		return fmt.Errorf("проверка лимита загрузок: %w", err)
	}
	if count >= s.maxUploadsHour {
		return fmt.Errorf("превышен лимит загрузок за час")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return fmt.Errorf("открытие файла: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("декодирование изображения: %w", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() > s.maxWidth || bounds.Dy() > s.maxHeight {
		slog.InfoContext(ctx, "ресайз изображения", "origin", bounds, "max", fmt.Sprintf("%dx%d", s.maxWidth, s.maxHeight))
		img = imaging.Resize(img, s.maxWidth, s.maxHeight, imaging.Lanczos)
	}

	buf := new(bytes.Buffer)
	if err := imaging.Encode(buf, img, imaging.PNG, imaging.PNGCompressionLevel(6)); err != nil {
		return fmt.Errorf("кодирование в PNG: %w", err)
	}

	imageData := buf.Bytes()
	imgUUID := uuid.New()
	s3Key := fmt.Sprintf("images/%s.png", imgUUID)

	objectName, err := s.minio.Upload(ctx, s3Key, bytes.NewReader(imageData), int64(len(imageData)), "image/png")
	if err != nil {
		return fmt.Errorf("загрузка в MinIO: %w", err)
	}

	width := img.Bounds().Max.X
	height := img.Bounds().Max.Y

	domainImg := domain.Image{
		ID:         imgUUID,
		FileName:   fileHeader.Filename,
		MimeType:   "image/png",
		SizeBytes:  int64(len(imageData)),
		Width:      &width,
		Height:     &height,
		S3Key:      objectName,
		BucketName: "edium-images",
	}

	if err := s.imageRepo.Save(ctx, domainImg); err != nil {
		return fmt.Errorf("сохранение в БД: %w", err)
	}

	slog.InfoContext(ctx, "изображение загружено", "id", imgUUID, "size", len(imageData), "dims", fmt.Sprintf("%dx%d", width, height))

	return nil
}

func (s *ImageService) Download(ctx context.Context, id uuid.UUID) (io.Reader, string, error) {
	domainImg, err := s.imageRepo.FindByID(ctx, id)
	if err != nil {
		return nil, "", fmt.Errorf("поиск изображения: %w", err)
	}
	if domainImg == nil {
		return nil, "", fmt.Errorf("изображение не найдено")
	}

	objectReader, err := s.minio.Download(ctx, domainImg.S3Key)
	if err != nil {
		return nil, "", fmt.Errorf("скачивание из MinIO: %w", err)
	}

	return objectReader, domainImg.MimeType, nil
}

func (s *ImageService) Delete(ctx context.Context, id uuid.UUID) error {
	domainImg, err := s.imageRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("поиск изображения: %w", err)
	}
	if domainImg == nil {
		return fmt.Errorf("изображение не найдено")
	}

	if err := s.imageRepo.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("мягкое удаление из БД: %w", err)
	}

	if err := s.minio.Delete(ctx, domainImg.S3Key); err != nil {
		slog.WarnContext(ctx, "не удалось удалить из MinIO", "err", err, "s3_key", domainImg.S3Key)
	}

	slog.InfoContext(ctx, "изображение удалено", "id", id)
	return nil
}
