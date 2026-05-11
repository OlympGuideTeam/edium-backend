package pushhandler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"herald/internal/domain"
	"herald/internal/middleware"
)

func init() { gin.SetMode(gin.TestMode) }

type mockPushService struct {
	registerErr      error
	deleteErr        error
	listResult       []domain.Notification
	listErr          error
	markReadErr      error
}

func (m *mockPushService) RegisterDevice(_ context.Context, _ uuid.UUID, _, _ string) error {
	return m.registerErr
}
func (m *mockPushService) DeleteDevice(_ context.Context, _ string) error { return m.deleteErr }
func (m *mockPushService) ListNotifications(_ context.Context, _ uuid.UUID) ([]domain.Notification, error) {
	return m.listResult, m.listErr
}
func (m *mockPushService) MarkRead(_ context.Context, _, _ uuid.UUID) error { return m.markReadErr }

func newRouter(svc PushService) *gin.Engine {
	r := gin.New()
	h := NewHandler(svc)
	h.Register(r.Group("/push"))
	return r
}

func withUser(r *http.Request, id uuid.UUID) *http.Request {
	return r.WithContext(middleware.WithUserID(r.Context(), id))
}

func getErrorCode(body *bytes.Buffer) string {
	var resp map[string]any
	_ = json.Unmarshal(body.Bytes(), &resp)
	code, _ := resp["error"].(string)
	return code
}

func TestRegisterDevice_NoUserID(t *testing.T) {
	r := newRouter(&mockPushService{})
	body, _ := json.Marshal(map[string]string{"fcm_token": "tok", "platform": "ios"})
	req := httptest.NewRequest(http.MethodPost, "/push/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRegisterDevice_BadJSON(t *testing.T) {
	r := newRouter(&mockPushService{})
	req := httptest.NewRequest(http.MethodPost, "/push/devices", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRegisterDevice_ServiceError(t *testing.T) {
	svc := &mockPushService{registerErr: errors.New("db error")}
	r := newRouter(svc)
	body, _ := json.Marshal(map[string]string{"fcm_token": "tok", "platform": "ios"})
	req := httptest.NewRequest(http.MethodPost, "/push/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestRegisterDevice_Success(t *testing.T) {
	r := newRouter(&mockPushService{})
	body, _ := json.Marshal(map[string]string{"fcm_token": "tok", "platform": "ios"})
	req := httptest.NewRequest(http.MethodPost, "/push/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUser(req, uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestDeleteDevice_NoUserID(t *testing.T) {
	r := newRouter(&mockPushService{})
	req := httptest.NewRequest(http.MethodDelete, "/push/devices/mytoken", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestDeleteDevice_Success(t *testing.T) {
	r := newRouter(&mockPushService{})
	req := httptest.NewRequest(http.MethodDelete, "/push/devices/mytoken", nil)
	req = withUser(req, uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestListNotifications_NoUserID(t *testing.T) {
	r := newRouter(&mockPushService{})
	req := httptest.NewRequest(http.MethodGet, "/push/notifications", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestListNotifications_Empty(t *testing.T) {
	r := newRouter(&mockPushService{})
	req := httptest.NewRequest(http.MethodGet, "/push/notifications", nil)
	req = withUser(req, uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp []any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("expected empty list")
	}
}

func TestListNotifications_WithData(t *testing.T) {
	route := "/test/123"
	role := "student"
	notifs := []domain.Notification{
		{
			ID:        uuid.New(),
			Title:     "Тест",
			Body:      "Результаты готовы",
			IsRead:    false,
			CreatedAt: time.Now(),
			Data:      &domain.NotificationData{Route: &route, Role: &role},
		},
	}
	r := newRouter(&mockPushService{listResult: notifs})
	req := httptest.NewRequest(http.MethodGet, "/push/notifications", nil)
	req = withUser(req, uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(resp))
	}
	if resp[0]["title"] != "Тест" {
		t.Errorf("unexpected title: %v", resp[0]["title"])
	}
}

func TestMarkRead_NoUserID(t *testing.T) {
	r := newRouter(&mockPushService{})
	req := httptest.NewRequest(http.MethodPatch, "/push/notifications/"+uuid.New().String()+"/read", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestMarkRead_InvalidUUID(t *testing.T) {
	r := newRouter(&mockPushService{})
	req := httptest.NewRequest(http.MethodPatch, "/push/notifications/not-a-uuid/read", nil)
	req = withUser(req, uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestMarkRead_ServiceError(t *testing.T) {
	svc := &mockPushService{markReadErr: errors.New("db error")}
	r := newRouter(svc)
	req := httptest.NewRequest(http.MethodPatch, "/push/notifications/"+uuid.New().String()+"/read", nil)
	req = withUser(req, uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestMarkRead_Success(t *testing.T) {
	r := newRouter(&mockPushService{})
	req := httptest.NewRequest(http.MethodPatch, "/push/notifications/"+uuid.New().String()+"/read", nil)
	req = withUser(req, uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}
