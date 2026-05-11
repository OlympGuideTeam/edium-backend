package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"louvre/internal/domain"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockService struct {
	uploadResult *domain.Image
	uploadErr    error
	downloadBody io.Reader
	downloadMime string
	downloadErr  error
	deleteErr    error
}

func (m *mockService) Upload(_ context.Context, _ *multipart.FileHeader) (*domain.Image, error) {
	return m.uploadResult, m.uploadErr
}
func (m *mockService) Download(_ context.Context, _ uuid.UUID) (io.Reader, string, error) {
	return m.downloadBody, m.downloadMime, m.downloadErr
}
func (m *mockService) Delete(_ context.Context, _ uuid.UUID) error { return m.deleteErr }

func newRouter(svc imageService) *gin.Engine {
	r := gin.New()
	h := NewHandler(svc)
	r.POST("/images/upload", h.Upload)
	r.GET("/images/:id", h.Download)
	r.DELETE("/images/:id", h.Delete)
	return r
}

func multipartBody(t *testing.T, filename, contentType string, data []byte) (*bytes.Buffer, string) {
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
	return body, w.FormDataContentType()
}

func getErrorCode(t *testing.T, body *bytes.Buffer) string {
	t.Helper()
	var resp map[string]any
	if err := json.Unmarshal(body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	code, _ := resp["error"].(string)
	return code
}

func TestUpload_MissingFile(t *testing.T) {
	r := newRouter(&mockService{})
	req := httptest.NewRequest(http.MethodPost, "/images/upload", nil)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xxx")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if code := getErrorCode(t, w.Body); code != "BAD_REQUEST" {
		t.Errorf("expected BAD_REQUEST, got %s", code)
	}
}

func TestUpload_FileTooLarge(t *testing.T) {
	svc := &mockService{uploadErr: errors.New("файл слишком большой, максимальный размер 5242880 байт")}
	r := newRouter(svc)

	body, ct := multipartBody(t, "img.png", "image/png", []byte("data"))
	req := httptest.NewRequest(http.MethodPost, "/images/upload", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if code := getErrorCode(t, w.Body); code != "FILE_TOO_LARGE" {
		t.Errorf("expected FILE_TOO_LARGE, got %s", code)
	}
}

func TestUpload_InvalidFileType(t *testing.T) {
	svc := &mockService{uploadErr: errors.New("невалидный тип файла: application/pdf")}
	r := newRouter(svc)

	body, ct := multipartBody(t, "doc.pdf", "application/pdf", []byte("data"))
	req := httptest.NewRequest(http.MethodPost, "/images/upload", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if code := getErrorCode(t, w.Body); code != "INVALID_FILE_TYPE" {
		t.Errorf("expected INVALID_FILE_TYPE, got %s", code)
	}
}

func TestUpload_TooManyUploads(t *testing.T) {
	svc := &mockService{uploadErr: errors.New("превышен лимит загрузок за час")}
	r := newRouter(svc)

	body, ct := multipartBody(t, "img.png", "image/png", []byte("data"))
	req := httptest.NewRequest(http.MethodPost, "/images/upload", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
	if code := getErrorCode(t, w.Body); code != "TOO_MANY_UPLOADS" {
		t.Errorf("expected TOO_MANY_UPLOADS, got %s", code)
	}
}

func TestUpload_InternalError(t *testing.T) {
	svc := &mockService{uploadErr: errors.New("unexpected db failure")}
	r := newRouter(svc)

	body, ct := multipartBody(t, "img.png", "image/png", []byte("data"))
	req := httptest.NewRequest(http.MethodPost, "/images/upload", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
	if code := getErrorCode(t, w.Body); code != "INTERNAL_ERROR" {
		t.Errorf("expected INTERNAL_ERROR, got %s", code)
	}
}

func TestUpload_Success(t *testing.T) {
	id := uuid.New()
	w2, h2 := 100, 200
	svc := &mockService{uploadResult: &domain.Image{
		ID:        id,
		FileName:  "photo.png",
		MimeType:  "image/png",
		SizeBytes: 1024,
		Width:     &w2,
		Height:    &h2,
	}}
	r := newRouter(svc)

	body, ct := multipartBody(t, "photo.png", "image/png", []byte("data"))
	req := httptest.NewRequest(http.MethodPost, "/images/upload", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["id"] != id.String() {
		t.Errorf("expected id %s, got %v", id, resp["id"])
	}
	if resp["mime_type"] != "image/png" {
		t.Errorf("unexpected mime_type: %v", resp["mime_type"])
	}
}

func TestDownload_InvalidUUID(t *testing.T) {
	r := newRouter(&mockService{})
	req := httptest.NewRequest(http.MethodGet, "/images/not-a-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDownload_NotFound(t *testing.T) {
	svc := &mockService{downloadErr: errors.New("изображение не найдено")}
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/images/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDownload_Success(t *testing.T) {
	content := strings.NewReader("imagedata")
	svc := &mockService{downloadBody: content, downloadMime: "image/png"}
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/images/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("expected image/png content-type, got %s", ct)
	}
	if w.Body.String() != "imagedata" {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
}

func TestDelete_InvalidUUID(t *testing.T) {
	r := newRouter(&mockService{})
	req := httptest.NewRequest(http.MethodDelete, "/images/bad-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDelete_Error(t *testing.T) {
	svc := &mockService{deleteErr: errors.New("not found")}
	r := newRouter(svc)

	req := httptest.NewRequest(http.MethodDelete, "/images/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestDelete_Success(t *testing.T) {
	r := newRouter(&mockService{})
	req := httptest.NewRequest(http.MethodDelete, "/images/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
