package otphandler

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

	"doorman/internal/domain"
	tokenhandler "doorman/internal/handler/token"
	"doorman/internal/pkg/apperr"
)

func init() { gin.SetMode(gin.TestMode) }

type mockOTPService struct {
	sendTTL    time.Duration
	sendErr    error
	verifyResult VerifyResult
	verifyErr  error
}

func (m *mockOTPService) SendOTP(_ context.Context, _ string, _ domain.Channel) (time.Duration, error) {
	return m.sendTTL, m.sendErr
}
func (m *mockOTPService) VerifyOTP(_ context.Context, _ string, _ uint64) (VerifyResult, error) {
	return m.verifyResult, m.verifyErr
}

func newRouter(svc IOTPService) *gin.Engine {
	r := gin.New()
	h := NewHandler(svc)
	r.POST("/otp/send", h.Send)
	r.POST("/otp/verify", h.Verify)
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

func getCode(w *httptest.ResponseRecorder) string {
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	code, _ := resp["error"].(string)
	return code
}

// --- Send ---

func TestSend_BadJSON(t *testing.T) {
	r := newRouter(&mockOTPService{})
	req := httptest.NewRequest(http.MethodPost, "/otp/send", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSend_OTPAlreadySent_WithRetryAfter(t *testing.T) {
	svc := &mockOTPService{sendTTL: 2 * time.Minute, sendErr: apperr.ErrOTPAlreadySent}
	r := newRouter(svc)
	w := postJSON(r, "/otp/send", map[string]string{"phone": "+71234567890", "channel": "tg"})

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if details, ok := resp["details"].(map[string]any); !ok || details["retry_after"] == nil {
		t.Error("expected retry_after in details")
	}
}

func TestSend_PhoneUnavailable(t *testing.T) {
	svc := &mockOTPService{sendErr: apperr.ErrPhoneUnavailable}
	r := newRouter(svc)
	w := postJSON(r, "/otp/send", map[string]string{"phone": "+71234567890", "channel": "tg"})

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestSend_InternalError(t *testing.T) {
	svc := &mockOTPService{sendErr: errors.New("db error")}
	r := newRouter(svc)
	w := postJSON(r, "/otp/send", map[string]string{"phone": "+71234567890", "channel": "tg"})

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestSend_Success(t *testing.T) {
	svc := &mockOTPService{sendTTL: 3 * time.Minute}
	r := newRouter(svc)
	w := postJSON(r, "/otp/send", map[string]string{"phone": "+71234567890", "channel": "sms"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["retry_after"] != float64(180) {
		t.Errorf("unexpected retry_after: %v", resp["retry_after"])
	}
}

// --- Verify ---

func TestVerify_BadJSON(t *testing.T) {
	r := newRouter(&mockOTPService{})
	req := httptest.NewRequest(http.MethodPost, "/otp/verify", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestVerify_ServiceError(t *testing.T) {
	svc := &mockOTPService{verifyErr: apperr.ErrOTPInvalid}
	r := newRouter(svc)
	w := postJSON(r, "/otp/verify", map[string]any{"phone": "+71234567890", "otp": 123456})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if code := getCode(w); code != "OTP_INVALID" {
		t.Errorf("expected OTP_INVALID, got %s", code)
	}
}

func TestVerify_ReturnsAuthTokens(t *testing.T) {
	tokens := &tokenhandler.AuthTokens{AccessToken: "acc", RefreshToken: "ref", ExpiresIn: 900}
	svc := &mockOTPService{verifyResult: tokens}
	r := newRouter(svc)
	w := postJSON(r, "/otp/verify", map[string]any{"phone": "+71234567890", "otp": 123456})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["access_token"] != "acc" {
		t.Errorf("unexpected access_token: %v", resp["access_token"])
	}
}

func TestVerify_ReturnsRegistrationToken(t *testing.T) {
	svc := &mockOTPService{verifyResult: &RegistrationToken{Token: "reg-tok-123"}}
	r := newRouter(svc)
	w := postJSON(r, "/otp/verify", map[string]any{"phone": "+71234567890", "otp": 123456})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["registration_token"] != "reg-tok-123" {
		t.Errorf("unexpected registration_token: %v", resp["registration_token"])
	}
}
