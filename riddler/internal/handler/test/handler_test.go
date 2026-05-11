package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/middleware"
	"riddler/internal/pkg/apperr"
)

func init() { gin.SetMode(gin.TestMode) }

type mockTestSvc struct {
	sessionID uuid.UUID
	createErr error
	finishErr error
}

func (m *mockTestSvc) CreateTestCourseSession(_ context.Context, _, _, _ uuid.UUID, _ domain.CreateTestCourseSessionParams) (uuid.UUID, error) {
	return m.sessionID, m.createErr
}
func (m *mockTestSvc) FinishTestCourseSession(_ context.Context, _, _ uuid.UUID) error {
	return m.finishErr
}

func newRouter(svc testService) *gin.Engine {
	r := gin.New()
	h := NewHandler(svc)
	r.POST("/sessions", h.CreateTestCourseSession)
	r.POST("/sessions/:session_id/finish", h.FinishTestCourseSession)
	return r
}

func withUser(req *http.Request, id uuid.UUID) *http.Request {
	return req.WithContext(middleware.WithUserID(req.Context(), id))
}

// --- CreateTestCourseSession ---

func TestCreateTestCourseSession_NoAuth(t *testing.T) {
	r := newRouter(&mockTestSvc{})
	body, _ := json.Marshal(map[string]string{
		"quiz_template_id": uuid.New().String(),
		"module_id":        uuid.New().String(),
	})
	req := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCreateTestCourseSession_InvalidQuizID(t *testing.T) {
	r := newRouter(&mockTestSvc{})
	body, _ := json.Marshal(map[string]string{
		"quiz_template_id": "bad-uuid",
		"module_id":        uuid.New().String(),
	})
	req := withUser(httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateTestCourseSession_QuizNotFound(t *testing.T) {
	r := newRouter(&mockTestSvc{createErr: apperr.ErrQuizNotFound})
	body, _ := json.Marshal(map[string]string{
		"quiz_template_id": uuid.New().String(),
		"module_id":        uuid.New().String(),
	})
	req := withUser(httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestCreateTestCourseSession_Success(t *testing.T) {
	id := uuid.New()
	r := newRouter(&mockTestSvc{sessionID: id})
	body, _ := json.Marshal(map[string]string{
		"quiz_template_id": uuid.New().String(),
		"module_id":        uuid.New().String(),
	})
	req := withUser(httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["session_id"] != id.String() {
		t.Errorf("unexpected session_id: %v", resp["session_id"])
	}
}

// --- FinishTestCourseSession ---

func TestFinishTestCourseSession_InvalidSessionID(t *testing.T) {
	r := newRouter(&mockTestSvc{})
	req := withUser(httptest.NewRequest(http.MethodPost, "/sessions/bad-uuid/finish", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestFinishTestCourseSession_SessionNotFound(t *testing.T) {
	r := newRouter(&mockTestSvc{finishErr: apperr.ErrSessionNotFound})
	req := withUser(httptest.NewRequest(http.MethodPost, "/sessions/"+uuid.New().String()+"/finish", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestFinishTestCourseSession_AlreadyFinished(t *testing.T) {
	r := newRouter(&mockTestSvc{finishErr: apperr.ErrSessionCompleted})
	req := withUser(httptest.NewRequest(http.MethodPost, "/sessions/"+uuid.New().String()+"/finish", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestFinishTestCourseSession_Success(t *testing.T) {
	r := newRouter(&mockTestSvc{})
	req := withUser(httptest.NewRequest(http.MethodPost, "/sessions/"+uuid.New().String()+"/finish", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
