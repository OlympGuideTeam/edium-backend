package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	stdpng "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/google/uuid"

	"louvre/internal/domain"
)

type mockRepo struct {
	saveErr       error
	findResult    *domain.Image
	findErr       error
	countResult   int
	countErr      error
	softDeleteErr error
}

func (m *mockRepo) Save(_ context.Context, _ domain.Image) error { return m.saveErr }
func (m *mockRepo) FindByID(_ context.Context, _ uuid.UUID) (*domain.Image, error) {
	return m.findResult, m.findErr
}
func (m *mockRepo) CountByUserID(_ context.Context, _ uuid.UUID) (int, error) {
	return m.countResult, m.countErr
}
func (m *mockRepo) SoftDelete(_ context.Context, _ uuid.UUID) error { return m.softDeleteErr }

type mockStorage struct {
	uploadKey    string
	uploadErr    error
	downloadBody io.ReadCloser
	downloadErr  error
	deleteErr    error
}

func (m *mockStorage) Upload(_ context.Context, objectName string, _ io.Reader, _ int64, _ string) (string, error) {
	if m.uploadErr != nil {
		return "", m.uploadErr
	}
	if m.uploadKey != "" {
		return m.uploadKey, nil
	}
	return objectName, nil
}
func (m *mockStorage) Download(_ context.Context, _ string) (io.ReadCloser, error) {
	return m.downloadBody, m.downloadErr
}
func (m *mockStorage) Delete(_ context.Context, _ string) error { return m.deleteErr }

func newService(repo *mockRepo, storage *mockStorage) *ImageService {
	return NewImageService(repo, storage, 5<<20, 4096, 4096, 10, []string{"image/png", "image/jpeg"})
}

func makePNG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	var buf bytes.Buffer
	_ = stdpng.Encode(&buf, img)
	return buf.Bytes()
}

func makeFileHeader(t *testing.T, filename, contentType string, data []byte) *multipart.FileHeader {
	t.Helper()
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf("form-data; name=\"file\"; filename=%q", filename))
	h.Set("Content-Type", contentType)
	pw, err := w.CreatePart(h)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = pw.Write(data)
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if err := req.ParseMultipartForm(10 << 20); err != nil {
		t.Fatal(err)
	}
	return req.MultipartForm.File["file"][0]
}

func TestUpload_FileTooLarge(t *testing.T) {
	svc := newService(&mockRepo{}, &mockStorage{})
	fh := makeFileHeader(t, "img.png", "image/png", makePNG(1, 1))
	fh.Size = 6 << 20

	_, err := svc.Upload(context.Background(), fh)
	if err == nil || !contains(err.Error(), "слишком большой") {
		t.Fatalf("expected file too large error, got %v", err)
	}
}

func TestUpload_InvalidMimeType(t *testing.T) {
	svc := newService(&mockRepo{}, &mockStorage{})
	fh := makeFileHeader(t, "doc.pdf", "application/pdf", []byte("pdf"))

	_, err := svc.Upload(context.Background(), fh)
	if err == nil || !contains(err.Error(), "невалидный тип файла") {
		t.Fatalf("expected invalid mime type error, got %v", err)
	}
}

func TestUpload_CountError(t *testing.T) {
	repo := &mockRepo{countErr: errors.New("db error")}
	svc := newService(repo, &mockStorage{})
	fh := makeFileHeader(t, "img.png", "image/png", makePNG(1, 1))

	_, err := svc.Upload(context.Background(), fh)
	if err == nil || !contains(err.Error(), "проверка лимита") {
		t.Fatalf("expected count error, got %v", err)
	}
}

func TestUpload_CountExceeded(t *testing.T) {
	repo := &mockRepo{countResult: 10}
	svc := newService(repo, &mockStorage{})
	fh := makeFileHeader(t, "img.png", "image/png", makePNG(1, 1))

	_, err := svc.Upload(context.Background(), fh)
	if err == nil || !contains(err.Error(), "лимит загрузок") {
		t.Fatalf("expected rate limit error, got %v", err)
	}
}

func TestUpload_InvalidImageData(t *testing.T) {
	svc := newService(&mockRepo{}, &mockStorage{})
	fh := makeFileHeader(t, "img.png", "image/png", []byte("not-an-image"))

	_, err := svc.Upload(context.Background(), fh)
	if err == nil || !contains(err.Error(), "декодирование") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestUpload_StorageError(t *testing.T) {
	storage := &mockStorage{uploadErr: errors.New("minio down")}
	svc := newService(&mockRepo{}, storage)
	fh := makeFileHeader(t, "img.png", "image/png", makePNG(10, 10))

	_, err := svc.Upload(context.Background(), fh)
	if err == nil || !contains(err.Error(), "загрузка в MinIO") {
		t.Fatalf("expected minio upload error, got %v", err)
	}
}

func TestUpload_RepoSaveError(t *testing.T) {
	repo := &mockRepo{saveErr: errors.New("insert failed")}
	svc := newService(repo, &mockStorage{})
	fh := makeFileHeader(t, "img.png", "image/png", makePNG(10, 10))

	_, err := svc.Upload(context.Background(), fh)
	if err == nil || !contains(err.Error(), "сохранение в БД") {
		t.Fatalf("expected repo save error, got %v", err)
	}
}

func TestUpload_Success(t *testing.T) {
	repo := &mockRepo{}
	storage := &mockStorage{}
	svc := newService(repo, storage)
	fh := makeFileHeader(t, "photo.png", "image/png", makePNG(100, 100))

	img, err := svc.Upload(context.Background(), fh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if img == nil {
		t.Fatal("expected non-nil image")
	}
	if img.MimeType != "image/png" {
		t.Errorf("expected image/png, got %s", img.MimeType)
	}
	if img.Width == nil || *img.Width != 100 {
		t.Errorf("unexpected width: %v", img.Width)
	}
}

func TestUpload_Resize(t *testing.T) {
	svc := newService(&mockRepo{}, &mockStorage{})
	fh := makeFileHeader(t, "big.png", "image/png", makePNG(8000, 8000))

	img, err := svc.Upload(context.Background(), fh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *img.Width > 4096 || *img.Height > 4096 {
		t.Errorf("image not resized: %dx%d", *img.Width, *img.Height)
	}
}

func TestDownload_RepoError(t *testing.T) {
	repo := &mockRepo{findErr: errors.New("db error")}
	svc := newService(repo, &mockStorage{})

	_, _, err := svc.Download(context.Background(), uuid.New())
	if err == nil || !contains(err.Error(), "поиск изображения") {
		t.Fatalf("expected find error, got %v", err)
	}
}

func TestDownload_NotFound(t *testing.T) {
	svc := newService(&mockRepo{}, &mockStorage{})

	_, _, err := svc.Download(context.Background(), uuid.New())
	if err == nil || !contains(err.Error(), "не найдено") {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestDownload_StorageError(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{findResult: &domain.Image{ID: id, S3Key: "images/x.png", MimeType: "image/png"}}
	storage := &mockStorage{downloadErr: errors.New("minio error")}
	svc := newService(repo, storage)

	_, _, err := svc.Download(context.Background(), id)
	if err == nil || !contains(err.Error(), "скачивание из MinIO") {
		t.Fatalf("expected minio error, got %v", err)
	}
}

func TestDownload_Success(t *testing.T) {
	id := uuid.New()
	content := io.NopCloser(bytes.NewReader([]byte("imagedata")))
	repo := &mockRepo{findResult: &domain.Image{ID: id, S3Key: "images/x.png", MimeType: "image/png"}}
	storage := &mockStorage{downloadBody: content}
	svc := newService(repo, storage)

	reader, mimeType, err := svc.Download(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mimeType != "image/png" {
		t.Errorf("expected image/png, got %s", mimeType)
	}
	data, _ := io.ReadAll(reader)
	if string(data) != "imagedata" {
		t.Errorf("unexpected content: %s", data)
	}
}

func TestDelete_RepoFindError(t *testing.T) {
	repo := &mockRepo{findErr: errors.New("db error")}
	svc := newService(repo, &mockStorage{})

	err := svc.Delete(context.Background(), uuid.New())
	if err == nil || !contains(err.Error(), "поиск изображения") {
		t.Fatalf("expected find error, got %v", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	svc := newService(&mockRepo{}, &mockStorage{})

	err := svc.Delete(context.Background(), uuid.New())
	if err == nil || !contains(err.Error(), "не найдено") {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestDelete_SoftDeleteError(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		findResult:    &domain.Image{ID: id, S3Key: "images/x.png"},
		softDeleteErr: errors.New("db error"),
	}
	svc := newService(repo, &mockStorage{})

	err := svc.Delete(context.Background(), id)
	if err == nil || !contains(err.Error(), "мягкое удаление") {
		t.Fatalf("expected soft delete error, got %v", err)
	}
}

func TestDelete_StorageErrorIsWarning(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{findResult: &domain.Image{ID: id, S3Key: "images/x.png"}}
	storage := &mockStorage{deleteErr: errors.New("minio error")}
	svc := newService(repo, storage)

	if err := svc.Delete(context.Background(), id); err != nil {
		t.Fatalf("minio delete error should not propagate, got %v", err)
	}
}

func TestDelete_Success(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{findResult: &domain.Image{ID: id, S3Key: "images/x.png"}}
	svc := newService(repo, &mockStorage{})

	if err := svc.Delete(context.Background(), id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
