package user

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

	"caesar/internal/domain"
	"caesar/internal/middleware"
	"caesar/internal/pkg/apperr"
)

func init() { gin.SetMode(gin.TestMode) }

type mockUserService struct {
	user        *domain.User
	userErr     error
	updateErr   error
	deleteErr   error
	statistic   *domain.UserStatistic
	statisticErr error
	rosterUsers []domain.User
	rosterErr   error
}

func (m *mockUserService) GetMe(_ context.Context, _ uuid.UUID) (*domain.User, error) {
	return m.user, m.userErr
}
func (m *mockUserService) UpdateMe(_ context.Context, _ uuid.UUID, _, _ *string) error {
	return m.updateErr
}
func (m *mockUserService) DeleteMe(_ context.Context, _ uuid.UUID) error { return m.deleteErr }
func (m *mockUserService) GetMeStatistic(_ context.Context, _ uuid.UUID) (*domain.UserStatistic, error) {
	return m.statistic, m.statisticErr
}
func (m *mockUserService) GetUsersRoster(_ context.Context, _ []uuid.UUID) ([]domain.User, error) {
	return m.rosterUsers, m.rosterErr
}

func newRouter(svc userService) *gin.Engine {
	r := gin.New()
	h := NewHandler(svc)
	r.GET("/users/me", h.GetMe)
	r.PATCH("/users/me", h.UpdateMe)
	r.DELETE("/users/me", h.DeleteMe)
	r.GET("/users/me/statistic", h.GetMeStatistic)
	r.POST("/users/roster", h.GetUsersRoster)
	return r
}

func withUser(req *http.Request, id uuid.UUID) *http.Request {
	return req.WithContext(middleware.WithUserID(req.Context(), id))
}

// --- GetMe ---

func TestGetMe_NoAuth(t *testing.T) {
	r := newRouter(&mockUserService{})
	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGetMe_NotFound(t *testing.T) {
	r := newRouter(&mockUserService{userErr: apperr.ErrUserNotFound})
	req := withUser(httptest.NewRequest(http.MethodGet, "/users/me", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetMe_Success(t *testing.T) {
	id := uuid.New()
	r := newRouter(&mockUserService{user: &domain.User{ID: id, Name: "Ivan", Surname: "Petrov"}})
	req := withUser(httptest.NewRequest(http.MethodGet, "/users/me", nil), id)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["name"] != "Ivan" {
		t.Errorf("unexpected name: %v", resp["name"])
	}
}

// --- UpdateMe ---

func TestUpdateMe_NoAuth(t *testing.T) {
	r := newRouter(&mockUserService{})
	body, _ := json.Marshal(map[string]string{"name": "Anna"})
	req := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestUpdateMe_NoFields(t *testing.T) {
	r := newRouter(&mockUserService{})
	body, _ := json.Marshal(map[string]any{})
	req := withUser(httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdateMe_Success(t *testing.T) {
	r := newRouter(&mockUserService{})
	body, _ := json.Marshal(map[string]string{"name": "Anna"})
	req := withUser(httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

// --- DeleteMe ---

func TestDeleteMe_Success(t *testing.T) {
	r := newRouter(&mockUserService{})
	req := withUser(httptest.NewRequest(http.MethodDelete, "/users/me", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestDeleteMe_Error(t *testing.T) {
	r := newRouter(&mockUserService{deleteErr: errors.New("db error")})
	req := withUser(httptest.NewRequest(http.MethodDelete, "/users/me", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- GetMeStatistic ---

func TestGetMeStatistic_Success(t *testing.T) {
	stat := &domain.UserStatistic{ClassTeacherCount: 3, StudentCount: 20}
	r := newRouter(&mockUserService{statistic: stat})
	req := withUser(httptest.NewRequest(http.MethodGet, "/users/me/statistic", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["class_teacher_count"] != float64(3) {
		t.Errorf("unexpected stat: %v", resp)
	}
}

// --- GetUsersRoster ---

func TestGetUsersRoster_BadJSON(t *testing.T) {
	r := newRouter(&mockUserService{})
	req := httptest.NewRequest(http.MethodPost, "/users/roster", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetUsersRoster_Success(t *testing.T) {
	id := uuid.New()
	users := []domain.User{{ID: id, Name: "Ivan"}}
	r := newRouter(&mockUserService{rosterUsers: users})
	body, _ := json.Marshal(map[string][]string{"user_ids": {id.String()}})
	req := httptest.NewRequest(http.MethodPost, "/users/roster", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	usersList, _ := resp["users"].([]any)
	if len(usersList) != 1 {
		t.Errorf("expected 1 user, got %d", len(usersList))
	}
}
