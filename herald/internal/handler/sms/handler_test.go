package smshandler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"herald/internal/domain"
)

func init() { gin.SetMode(gin.TestMode) }

type mockSMSTasks struct {
	listResult []domain.SMSTask
	listErr    error
	ackErr     error
}

func (m *mockSMSTasks) ListPending(_ context.Context, _ int) ([]domain.SMSTask, error) {
	return m.listResult, m.listErr
}
func (m *mockSMSTasks) Ack(_ context.Context, _ uuid.UUID, _ bool, _ string) error {
	return m.ackErr
}

const testAPIKey = "secret-key"

func newRouter(tasks SMSTaskRepository) *gin.Engine {
	r := gin.New()
	h := NewHandler(tasks, testAPIKey)
	h.Register(r.Group(""))
	return r
}

func TestListTasks_Unauthorized(t *testing.T) {
	r := newRouter(&mockSMSTasks{})
	req := httptest.NewRequest(http.MethodGet, "/sms/tasks", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestListTasks_WrongKey(t *testing.T) {
	r := newRouter(&mockSMSTasks{})
	req := httptest.NewRequest(http.MethodGet, "/sms/tasks", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestListTasks_RepoError(t *testing.T) {
	r := newRouter(&mockSMSTasks{listErr: errors.New("db error")})
	req := httptest.NewRequest(http.MethodGet, "/sms/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestListTasks_Empty(t *testing.T) {
	r := newRouter(&mockSMSTasks{})
	req := httptest.NewRequest(http.MethodGet, "/sms/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp []any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp) != 0 {
		t.Errorf("expected empty list")
	}
}

func TestListTasks_WithItems(t *testing.T) {
	tasks := []domain.SMSTask{
		{ID: uuid.New(), Phone: "+79001234567", Text: "Ваш код: 123456"},
	}
	r := newRouter(&mockSMSTasks{listResult: tasks})
	req := httptest.NewRequest(http.MethodGet, "/sms/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp []map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp) != 1 {
		t.Fatalf("expected 1 task, got %d", len(resp))
	}
	if resp[0]["phone"] != "+79001234567" {
		t.Errorf("unexpected phone: %v", resp[0]["phone"])
	}
}

func TestAckTask_Unauthorized(t *testing.T) {
	r := newRouter(&mockSMSTasks{})
	body, _ := json.Marshal(map[string]any{"success": true})
	req := httptest.NewRequest(http.MethodPost, "/sms/tasks/"+uuid.New().String()+"/ack", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAckTask_InvalidUUID(t *testing.T) {
	r := newRouter(&mockSMSTasks{})
	body, _ := json.Marshal(map[string]any{"success": true})
	req := httptest.NewRequest(http.MethodPost, "/sms/tasks/bad-uuid/ack", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAckTask_BadJSON(t *testing.T) {
	r := newRouter(&mockSMSTasks{})
	req := httptest.NewRequest(http.MethodPost, "/sms/tasks/"+uuid.New().String()+"/ack", bytes.NewReader([]byte("bad")))
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAckTask_RepoError(t *testing.T) {
	r := newRouter(&mockSMSTasks{ackErr: errors.New("db error")})
	body, _ := json.Marshal(map[string]any{"success": false, "error": "send failed"})
	req := httptest.NewRequest(http.MethodPost, "/sms/tasks/"+uuid.New().String()+"/ack", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestAckTask_Success(t *testing.T) {
	r := newRouter(&mockSMSTasks{})
	body, _ := json.Marshal(map[string]any{"success": true})
	req := httptest.NewRequest(http.MethodPost, "/sms/tasks/"+uuid.New().String()+"/ack", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}
