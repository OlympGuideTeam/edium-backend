package reghandler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	tokenhandler "doorman/internal/handler/token"
	"doorman/internal/pkg/apperr"
)

func init() { gin.SetMode(gin.TestMode) }

type mockRegService struct {
	tokens *tokenhandler.AuthTokens
	err    error
}

func (m *mockRegService) Register(_ context.Context, _, _, _, _ string) (*tokenhandler.AuthTokens, error) {
	return m.tokens, m.err
}

func newRouter(svc IRegistrationService) *gin.Engine {
	r := gin.New()
	h := NewHandler(svc)
	r.POST("/register", h.Register)
	return r
}

func postWithHeader(r *gin.Engine, regToken string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if regToken != "" {
		req.Header.Set("X-Reg-Token", regToken)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRegister_MissingRegToken(t *testing.T) {
	r := newRouter(&mockRegService{})
	w := postWithHeader(r, "", map[string]string{"phone": "+71234567890", "name": "Ivan", "surname": "Petrov"})

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "MISSING_REG_TOKEN" {
		t.Errorf("expected MISSING_REG_TOKEN, got %v", resp["error"])
	}
}

func TestRegister_BadJSON(t *testing.T) {
	r := newRouter(&mockRegService{})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Reg-Token", "tok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRegister_InvalidRegToken(t *testing.T) {
	svc := &mockRegService{err: apperr.ErrRegTokenInvalid}
	r := newRouter(svc)
	w := postWithHeader(r, "wrong", map[string]string{"phone": "+71234567890", "name": "Ivan", "surname": "Petrov"})

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRegister_InternalError(t *testing.T) {
	svc := &mockRegService{err: errors.New("db error")}
	r := newRouter(svc)
	w := postWithHeader(r, "tok", map[string]string{"phone": "+71234567890", "name": "Ivan", "surname": "Petrov"})

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestRegister_Success(t *testing.T) {
	tokens := &tokenhandler.AuthTokens{AccessToken: "acc", RefreshToken: "ref", ExpiresIn: 900}
	svc := &mockRegService{tokens: tokens}
	r := newRouter(svc)
	w := postWithHeader(r, "valid-tok", map[string]string{"phone": "+71234567890", "name": "Ivan", "surname": "Petrov"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["access_token"] != "acc" {
		t.Errorf("unexpected access_token: %v", resp["access_token"])
	}
}
