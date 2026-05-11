package tokenhandler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"doorman/internal/pkg/apperr"
)

func init() { gin.SetMode(gin.TestMode) }

type mockTokenService struct {
	refreshResult *AuthTokens
	refreshErr    error
	logoutErr     error
}

func (m *mockTokenService) Refresh(_ context.Context, _ string) (*AuthTokens, error) {
	return m.refreshResult, m.refreshErr
}
func (m *mockTokenService) Logout(_ context.Context, _ string) error { return m.logoutErr }

func newRouter(svc ITokenService) *gin.Engine {
	r := gin.New()
	h := NewHandler(svc)
	r.POST("/token/refresh", h.Refresh)
	r.POST("/token/logout", h.Logout)
	return r
}

func postJSON(r *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// --- Refresh ---

func TestRefresh_BadJSON(t *testing.T) {
	r := newRouter(&mockTokenService{})
	req := httptest.NewRequest(http.MethodPost, "/token/refresh", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	svc := &mockTokenService{refreshErr: apperr.ErrRefreshTokenInvalid}
	r := newRouter(svc)
	w := postJSON(r, "/token/refresh", map[string]string{"refresh_token": "bad"})

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRefresh_InternalError(t *testing.T) {
	svc := &mockTokenService{refreshErr: errors.New("db error")}
	r := newRouter(svc)
	w := postJSON(r, "/token/refresh", map[string]string{"refresh_token": "tok"})

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestRefresh_Success(t *testing.T) {
	tokens := &AuthTokens{AccessToken: "new-acc", RefreshToken: "new-ref", ExpiresIn: 900}
	svc := &mockTokenService{refreshResult: tokens}
	r := newRouter(svc)
	w := postJSON(r, "/token/refresh", map[string]string{"refresh_token": "valid"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["access_token"] != "new-acc" {
		t.Errorf("unexpected access_token: %v", resp["access_token"])
	}
}

// --- Logout ---

func TestLogout_BadJSON(t *testing.T) {
	r := newRouter(&mockTokenService{})
	req := httptest.NewRequest(http.MethodPost, "/token/logout", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestLogout_ServiceError(t *testing.T) {
	svc := &mockTokenService{logoutErr: apperr.ErrRefreshTokenInvalid}
	r := newRouter(svc)
	w := postJSON(r, "/token/logout", map[string]string{"refresh_token": "bad"})

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestLogout_Success(t *testing.T) {
	r := newRouter(&mockTokenService{})
	w := postJSON(r, "/token/logout", map[string]string{"refresh_token": "valid"})

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}
