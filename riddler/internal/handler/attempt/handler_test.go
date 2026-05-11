package attempt

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

	"riddler/internal/domain"
	"riddler/internal/middleware"
	"riddler/internal/pkg/apperr"
)

func init() { gin.SetMode(gin.TestMode) }

type mockAttemptSvc struct {
	attempt          *domain.Attempt
	questions        []domain.QuestionForStudent
	createErr        error
	submitErr        error
	finishErr        error
	sessionAttempts  []domain.AttemptSummary
	listErr          error
	reviewAttempt    *domain.Attempt
	reviewAnswers    []domain.AnswerWithQuestion
	reviewEnriched   bool
	reviewErr        error
	gradeErr         error
	publishErr       error
	awaitingSessions []domain.AwaitingReviewSession
	awaitingErr      error
	dashboard        *domain.StudentDashboard
	dashboardErr     error
	statistic        *domain.UserStatistic
	statisticErr     error
}

func (m *mockAttemptSvc) Create(_ context.Context, _, _ uuid.UUID) (*domain.Attempt, []domain.QuestionForStudent, error) {
	return m.attempt, m.questions, m.createErr
}
func (m *mockAttemptSvc) SubmitAnswer(_ context.Context, _, _, _ uuid.UUID, _ map[string]any) error {
	return m.submitErr
}
func (m *mockAttemptSvc) Finish(_ context.Context, _, _ uuid.UUID) error { return m.finishErr }
func (m *mockAttemptSvc) ListSessionAttempts(_ context.Context, _, _ uuid.UUID) ([]domain.AttemptSummary, error) {
	return m.sessionAttempts, m.listErr
}
func (m *mockAttemptSvc) GetAttemptReview(_ context.Context, _ uuid.UUID, _ *uuid.UUID) (*domain.Attempt, []domain.AnswerWithQuestion, bool, error) {
	return m.reviewAttempt, m.reviewAnswers, m.reviewEnriched, m.reviewErr
}
func (m *mockAttemptSvc) GradeAttempt(_ context.Context, _, _ uuid.UUID, _ []domain.GradeItem) error {
	return m.gradeErr
}
func (m *mockAttemptSvc) PublishSession(_ context.Context, _, _ uuid.UUID) error {
	return m.publishErr
}
func (m *mockAttemptSvc) ListAwaitingReview(_ context.Context, _ uuid.UUID) ([]domain.AwaitingReviewSession, error) {
	return m.awaitingSessions, m.awaitingErr
}
func (m *mockAttemptSvc) GetStudentDashboard(_ context.Context, _ uuid.UUID) (*domain.StudentDashboard, error) {
	return m.dashboard, m.dashboardErr
}
func (m *mockAttemptSvc) GetUserStatistic(_ context.Context, _ uuid.UUID) (*domain.UserStatistic, error) {
	return m.statistic, m.statisticErr
}

func newRouter(svc attemptService) *gin.Engine {
	r := gin.New()
	h := NewHandler(svc)
	r.POST("/sessions/:session_id/attempts", h.CreateAttempt)
	r.POST("/attempts/:attempt_id/answers", h.SubmitAnswer)
	r.POST("/attempts/:attempt_id/finish", h.Finish)
	r.GET("/sessions/:session_id/attempts", h.ListSessionAttempts)
	r.GET("/attempts/:attempt_id/review", h.GetAttemptReview)
	r.POST("/attempts/:attempt_id/grade", h.GradeAttempt)
	r.POST("/sessions/:session_id/publish", h.PublishSession)
	r.GET("/reviews/awaiting", h.ListAwaitingReview)
	r.GET("/dashboard", h.GetStudentDashboard)
	r.GET("/me/statistic", h.GetMeStatistic)
	return r
}

func withUser(req *http.Request, id uuid.UUID) *http.Request {
	return req.WithContext(middleware.WithUserID(req.Context(), id))
}

// --- CreateAttempt ---

func TestCreateAttempt_InvalidSessionID(t *testing.T) {
	r := newRouter(&mockAttemptSvc{})
	req := withUser(httptest.NewRequest(http.MethodPost, "/sessions/bad-uuid/attempts", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateAttempt_SessionNotFound(t *testing.T) {
	r := newRouter(&mockAttemptSvc{createErr: apperr.ErrSessionNotFound})
	req := withUser(httptest.NewRequest(http.MethodPost, "/sessions/"+uuid.New().String()+"/attempts", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestCreateAttempt_AlreadyExists(t *testing.T) {
	r := newRouter(&mockAttemptSvc{createErr: apperr.ErrAttemptAlreadyExists})
	req := withUser(httptest.NewRequest(http.MethodPost, "/sessions/"+uuid.New().String()+"/attempts", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestCreateAttempt_Success(t *testing.T) {
	attempt := &domain.Attempt{ID: uuid.New()}
	r := newRouter(&mockAttemptSvc{attempt: attempt, questions: []domain.QuestionForStudent{}})
	req := withUser(httptest.NewRequest(http.MethodPost, "/sessions/"+uuid.New().String()+"/attempts", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
}

// --- SubmitAnswer ---

func TestSubmitAnswer_InvalidAttemptID(t *testing.T) {
	r := newRouter(&mockAttemptSvc{})
	body, _ := json.Marshal(map[string]any{"question_id": uuid.New().String(), "answer_data": map[string]any{}})
	req := withUser(httptest.NewRequest(http.MethodPost, "/attempts/bad-uuid/answers", bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSubmitAnswer_AttemptNotFound(t *testing.T) {
	r := newRouter(&mockAttemptSvc{submitErr: apperr.ErrAttemptNotFound})
	body, _ := json.Marshal(map[string]any{"question_id": uuid.New().String(), "answer_data": map[string]any{}})
	req := withUser(httptest.NewRequest(http.MethodPost, "/attempts/"+uuid.New().String()+"/answers", bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestSubmitAnswer_Success(t *testing.T) {
	r := newRouter(&mockAttemptSvc{})
	body, _ := json.Marshal(map[string]any{"question_id": uuid.New().String(), "answer_data": map[string]any{"text": "answer"}})
	req := withUser(httptest.NewRequest(http.MethodPost, "/attempts/"+uuid.New().String()+"/answers", bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- Finish ---

func TestFinish_AttemptNotActive(t *testing.T) {
	r := newRouter(&mockAttemptSvc{finishErr: apperr.ErrAttemptNotActive})
	req := withUser(httptest.NewRequest(http.MethodPost, "/attempts/"+uuid.New().String()+"/finish", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestFinish_Success(t *testing.T) {
	r := newRouter(&mockAttemptSvc{})
	req := withUser(httptest.NewRequest(http.MethodPost, "/attempts/"+uuid.New().String()+"/finish", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- ListSessionAttempts ---

func TestListSessionAttempts_Success(t *testing.T) {
	attempts := []domain.AttemptSummary{{ID: uuid.New(), Status: domain.AttemptStatusCompleted}}
	r := newRouter(&mockAttemptSvc{sessionAttempts: attempts})
	req := withUser(httptest.NewRequest(http.MethodGet, "/sessions/"+uuid.New().String()+"/attempts", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp []any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp) != 1 {
		t.Errorf("expected 1 attempt, got %d", len(resp))
	}
}

// --- GetAttemptReview ---

func TestGetAttemptReview_InvalidID(t *testing.T) {
	r := newRouter(&mockAttemptSvc{})
	req := httptest.NewRequest(http.MethodGet, "/attempts/bad-uuid/review", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetAttemptReview_NotFound(t *testing.T) {
	r := newRouter(&mockAttemptSvc{reviewErr: apperr.ErrAttemptNotFound})
	req := httptest.NewRequest(http.MethodGet, "/attempts/"+uuid.New().String()+"/review", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetAttemptReview_Success(t *testing.T) {
	now := time.Now()
	attempt := &domain.Attempt{ID: uuid.New(), UserID: uuid.New(), Status: domain.AttemptStatusCompleted, StartedAt: now}
	r := newRouter(&mockAttemptSvc{reviewAttempt: attempt})
	req := httptest.NewRequest(http.MethodGet, "/attempts/"+uuid.New().String()+"/review", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- GradeAttempt ---

func TestGradeAttempt_InvalidID(t *testing.T) {
	r := newRouter(&mockAttemptSvc{})
	body, _ := json.Marshal(map[string]any{"grades": []any{}})
	req := withUser(httptest.NewRequest(http.MethodPost, "/attempts/bad-uuid/grade", bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGradeAttempt_Success(t *testing.T) {
	r := newRouter(&mockAttemptSvc{})
	body, _ := json.Marshal(map[string]any{"grades": []map[string]any{
		{"submission_id": uuid.New().String(), "score": 8.0},
	}})
	req := withUser(httptest.NewRequest(http.MethodPost, "/attempts/"+uuid.New().String()+"/grade", bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGradeAttempt_ScoreInvalid(t *testing.T) {
	r := newRouter(&mockAttemptSvc{gradeErr: apperr.ErrScoreInvalid})
	body, _ := json.Marshal(map[string]any{"grades": []map[string]any{
		{"submission_id": uuid.New().String(), "score": -1.0},
	}})
	req := withUser(httptest.NewRequest(http.MethodPost, "/attempts/"+uuid.New().String()+"/grade", bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}

// --- PublishSession ---

func TestPublishSession_Success(t *testing.T) {
	r := newRouter(&mockAttemptSvc{})
	req := withUser(httptest.NewRequest(http.MethodPost, "/sessions/"+uuid.New().String()+"/publish", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestPublishSession_NotCompleted(t *testing.T) {
	r := newRouter(&mockAttemptSvc{publishErr: apperr.ErrAttemptNotCompleted})
	req := withUser(httptest.NewRequest(http.MethodPost, "/sessions/"+uuid.New().String()+"/publish", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

// --- ListAwaitingReview ---

func TestListAwaitingReview_Success(t *testing.T) {
	sessions := []domain.AwaitingReviewSession{{SessionID: uuid.New()}}
	r := newRouter(&mockAttemptSvc{awaitingSessions: sessions})
	req := withUser(httptest.NewRequest(http.MethodGet, "/reviews/awaiting", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- GetStudentDashboard ---

func TestGetStudentDashboard_Success(t *testing.T) {
	dashboard := &domain.StudentDashboard{
		RecentGrades: []domain.RecentGradeItem{},
		ActiveTests:  []domain.ActiveTestItem{},
	}
	r := newRouter(&mockAttemptSvc{dashboard: dashboard})
	req := withUser(httptest.NewRequest(http.MethodGet, "/dashboard", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetStudentDashboard_Error(t *testing.T) {
	r := newRouter(&mockAttemptSvc{dashboardErr: errors.New("db error")})
	req := withUser(httptest.NewRequest(http.MethodGet, "/dashboard", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- GetMeStatistic ---

func TestGetMeStatistic_Success(t *testing.T) {
	stat := &domain.UserStatistic{QuizCountPassed: 3}
	r := newRouter(&mockAttemptSvc{statistic: stat})
	req := withUser(httptest.NewRequest(http.MethodGet, "/me/statistic", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
