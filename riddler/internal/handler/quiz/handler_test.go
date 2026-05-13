package quiz

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

	"riddler/internal/domain"
	"riddler/internal/middleware"
	"riddler/internal/pkg/apperr"
)

func init() { gin.SetMode(gin.TestMode) }

type mockQuizSvc struct {
	createID          uuid.UUID
	createErr         error
	quizDetail        *domain.QuizDetail
	quizErr           error
	studentView       *domain.QuizStudentView
	studentViewErr    error
	updateErr         error
	deleteErr         error
	publishErr        error
	copyID            uuid.UUID
	copyErr           error
	listItems         []domain.QuizListItem
	listErr           error
	addQuestionID     uuid.UUID
	addQuestionIdx    int
	addQuestionErr    error
	deleteQuestionErr error
	reorderErr        error
	generateJobID     uuid.UUID
	generateErr       error
	deleteSessionErr  error
	createTestIDs     [2]uuid.UUID
	createTestErr     error
	createLiveIDs     [2]uuid.UUID
	createLiveErr     error
}

func (m *mockQuizSvc) CreateQuiz(_ context.Context, _ uuid.UUID, _ string, _ *string, _ domain.QuizDefaultSettings, _ *uuid.UUID) (uuid.UUID, error) {
	return m.createID, m.createErr
}
func (m *mockQuizSvc) GetQuiz(_ context.Context, _, _ uuid.UUID) (*domain.QuizDetail, error) {
	return m.quizDetail, m.quizErr
}
func (m *mockQuizSvc) GetQuizForStudent(_ context.Context, _, _ uuid.UUID) (*domain.QuizStudentView, error) {
	return m.studentView, m.studentViewErr
}
func (m *mockQuizSvc) UpdateQuiz(_ context.Context, _, _ uuid.UUID, _, _ *string, _ *domain.QuizDefaultSettings) error {
	return m.updateErr
}
func (m *mockQuizSvc) DeleteQuiz(_ context.Context, _, _ uuid.UUID) error  { return m.deleteErr }
func (m *mockQuizSvc) PublishQuiz(_ context.Context, _, _ uuid.UUID) error { return m.publishErr }
func (m *mockQuizSvc) CopyQuiz(_ context.Context, _, _ uuid.UUID, _ *uuid.UUID) (uuid.UUID, error) {
	return m.copyID, m.copyErr
}
func (m *mockQuizSvc) ListQuizzes(_ context.Context, _ domain.Role, _ *uuid.UUID, _ *string) ([]domain.QuizListItem, error) {
	return m.listItems, m.listErr
}
func (m *mockQuizSvc) ListMyQuizzes(_ context.Context, _ uuid.UUID, _ *string) ([]domain.QuizListItem, error) {
	return m.listItems, m.listErr
}
func (m *mockQuizSvc) AddQuestion(_ context.Context, _, _ uuid.UUID, _ domain.AddQuestionParams) (uuid.UUID, int, error) {
	return m.addQuestionID, m.addQuestionIdx, m.addQuestionErr
}
func (m *mockQuizSvc) DeleteQuestion(_ context.Context, _, _, _ uuid.UUID) error {
	return m.deleteQuestionErr
}
func (m *mockQuizSvc) ReorderQuestions(_ context.Context, _, _ uuid.UUID, _ []uuid.UUID) error {
	return m.reorderErr
}
func (m *mockQuizSvc) GenerateQuestions(_ context.Context, _, _ uuid.UUID, _ string) (uuid.UUID, error) {
	return m.generateJobID, m.generateErr
}
func (m *mockQuizSvc) DeleteCourseSession(_ context.Context, _, _ uuid.UUID) error {
	return m.deleteSessionErr
}
func (m *mockQuizSvc) CreateTestCourseSessionInline(_ context.Context, _ uuid.UUID, _ domain.CreateTestCourseSessionInlineParams) (uuid.UUID, uuid.UUID, error) {
	return m.createTestIDs[0], m.createTestIDs[1], m.createTestErr
}
func (m *mockQuizSvc) CreateLiveCourseSessionInline(_ context.Context, _ uuid.UUID, _ domain.CreateLiveCourseSessionInlineParams) (uuid.UUID, uuid.UUID, error) {
	return m.createLiveIDs[0], m.createLiveIDs[1], m.createLiveErr
}

func newRouter(svc quizService) *gin.Engine {
	r := gin.New()
	h := NewHandler(svc)
	r.POST("/quizzes", h.CreateQuiz)
	r.GET("/quizzes/:id", h.GetQuiz)
	r.PATCH("/quizzes/:id", h.UpdateQuiz)
	r.DELETE("/quizzes/:id", h.DeleteQuiz)
	r.POST("/quizzes/:id/publish", h.PublishQuiz)
	r.POST("/quizzes/:id/copy", h.CopyQuiz)
	r.GET("/quizzes", h.ListQuizzes)
	r.GET("/quizzes/my", h.ListMyQuizzes)
	r.POST("/quizzes/:id/questions", h.AddQuestion)
	r.DELETE("/quizzes/:id/questions/:question_id", h.DeleteQuestion)
	r.PUT("/quizzes/:id/questions/reorder", h.ReorderQuestions)
	r.POST("/quizzes/:id/generate", h.GenerateQuestions)
	r.DELETE("/sessions/:session_id", h.DeleteCourseSession)
	return r
}

func withUser(req *http.Request, id uuid.UUID) *http.Request {
	return req.WithContext(middleware.WithUserID(req.Context(), id))
}

func postJSON(r *gin.Engine, path string, body any, userID *uuid.UUID) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if userID != nil {
		req = withUser(req, *userID)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// --- CreateQuiz ---

func TestCreateQuiz_NoAuth(t *testing.T) {
	r := newRouter(&mockQuizSvc{})
	w := postJSON(r, "/quizzes", map[string]string{"title": "Math"}, nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCreateQuiz_Success(t *testing.T) {
	id := uuid.New()
	uid := uuid.New()
	r := newRouter(&mockQuizSvc{createID: id})
	w := postJSON(r, "/quizzes", map[string]string{"title": "Math"}, &uid)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateQuiz_EmptyTitle(t *testing.T) {
	uid := uuid.New()
	r := newRouter(&mockQuizSvc{createErr: apperr.ErrQuizEmptyTitle})
	w := postJSON(r, "/quizzes", map[string]string{"title": "   "}, &uid)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}

// --- GetQuiz ---

func TestGetQuiz_InvalidUUID(t *testing.T) {
	uid := uuid.New()
	r := newRouter(&mockQuizSvc{})
	req := withUser(httptest.NewRequest(http.MethodGet, "/quizzes/bad-uuid?role=teacher", nil), uid)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetQuiz_InvalidRole(t *testing.T) {
	uid := uuid.New()
	r := newRouter(&mockQuizSvc{})
	req := withUser(httptest.NewRequest(http.MethodGet, "/quizzes/"+uuid.New().String()+"?role=admin", nil), uid)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetQuiz_AsTeacher_Success(t *testing.T) {
	uid := uuid.New()
	detail := &domain.QuizDetail{QuizTemplate: domain.QuizTemplate{ID: uuid.New(), Title: "Math"}}
	r := newRouter(&mockQuizSvc{quizDetail: detail})
	req := withUser(httptest.NewRequest(http.MethodGet, "/quizzes/"+uuid.New().String()+"?role=teacher", nil), uid)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetQuiz_AsStudent_Success(t *testing.T) {
	uid := uuid.New()
	view := &domain.QuizStudentView{ID: uuid.New(), Title: "Math"}
	r := newRouter(&mockQuizSvc{studentView: view})
	req := withUser(httptest.NewRequest(http.MethodGet, "/quizzes/"+uuid.New().String()+"?role=student", nil), uid)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetQuiz_NotFound(t *testing.T) {
	uid := uuid.New()
	r := newRouter(&mockQuizSvc{quizErr: apperr.ErrQuizNotFound})
	req := withUser(httptest.NewRequest(http.MethodGet, "/quizzes/"+uuid.New().String()+"?role=teacher", nil), uid)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- DeleteQuiz ---

func TestDeleteQuiz_Success(t *testing.T) {
	uid := uuid.New()
	r := newRouter(&mockQuizSvc{})
	req := withUser(httptest.NewRequest(http.MethodDelete, "/quizzes/"+uuid.New().String(), nil), uid)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestDeleteQuiz_Forbidden(t *testing.T) {
	uid := uuid.New()
	r := newRouter(&mockQuizSvc{deleteErr: apperr.ErrQuizForbidden})
	req := withUser(httptest.NewRequest(http.MethodDelete, "/quizzes/"+uuid.New().String(), nil), uid)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

// --- UpdateQuiz ---

func TestUpdateQuiz_Success(t *testing.T) {
	uid := uuid.New()
	r := newRouter(&mockQuizSvc{})
	body, _ := json.Marshal(map[string]string{"title": "New"})
	req := withUser(httptest.NewRequest(http.MethodPatch, "/quizzes/"+uuid.New().String(), bytes.NewReader(body)), uid)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- PublishQuiz ---

func TestPublishQuiz_AlreadyPublished(t *testing.T) {
	uid := uuid.New()
	r := newRouter(&mockQuizSvc{publishErr: apperr.ErrQuizAlreadyPublished})
	req := withUser(httptest.NewRequest(http.MethodPost, "/quizzes/"+uuid.New().String()+"/publish", nil), uid)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestPublishQuiz_Success(t *testing.T) {
	uid := uuid.New()
	r := newRouter(&mockQuizSvc{})
	req := withUser(httptest.NewRequest(http.MethodPost, "/quizzes/"+uuid.New().String()+"/publish", nil), uid)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- ListQuizzes ---

func TestListQuizzes_InvalidRole(t *testing.T) {
	uid := uuid.New()
	r := newRouter(&mockQuizSvc{})
	req := withUser(httptest.NewRequest(http.MethodGet, "/quizzes?role=admin", nil), uid)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestListQuizzes_Success(t *testing.T) {
	uid := uuid.New()
	items := []domain.QuizListItem{{ID: uuid.New(), Title: "Math"}}
	r := newRouter(&mockQuizSvc{listItems: items})
	req := withUser(httptest.NewRequest(http.MethodGet, "/quizzes?role=teacher", nil), uid)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- ListMyQuizzes ---

func TestListMyQuizzes_Success(t *testing.T) {
	uid := uuid.New()
	r := newRouter(&mockQuizSvc{listItems: []domain.QuizListItem{}})
	req := withUser(httptest.NewRequest(http.MethodGet, "/quizzes/my", nil), uid)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- AddQuestion ---

func TestAddQuestion_Success(t *testing.T) {
	uid := uuid.New()
	id := uuid.New()
	r := newRouter(&mockQuizSvc{addQuestionID: id, addQuestionIdx: 0})
	body, _ := json.Marshal(map[string]any{
		"type": "single_choice",
		"text": "Question?",
		"answer_options": []map[string]any{
			{"text": "A", "is_correct": true},
			{"text": "B", "is_correct": false},
		},
	})
	req := withUser(httptest.NewRequest(http.MethodPost, "/quizzes/"+uuid.New().String()+"/questions", bytes.NewReader(body)), uid)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAddQuestion_ValidationError(t *testing.T) {
	uid := uuid.New()
	r := newRouter(&mockQuizSvc{addQuestionErr: apperr.ErrQuestionOneCorrect})
	body, _ := json.Marshal(map[string]any{
		"type": "single_choice",
		"text": "Q?",
		"answer_options": []map[string]any{
			{"text": "A", "is_correct": false},
			{"text": "B", "is_correct": false},
		},
	})
	req := withUser(httptest.NewRequest(http.MethodPost, "/quizzes/"+uuid.New().String()+"/questions", bytes.NewReader(body)), uid)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}

// --- DeleteQuestion ---

func TestDeleteQuestion_InvalidQuestionID(t *testing.T) {
	uid := uuid.New()
	r := newRouter(&mockQuizSvc{})
	req := withUser(httptest.NewRequest(http.MethodDelete, "/quizzes/"+uuid.New().String()+"/questions/bad-uuid", nil), uid)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDeleteQuestion_Success(t *testing.T) {
	uid := uuid.New()
	r := newRouter(&mockQuizSvc{})
	req := withUser(httptest.NewRequest(http.MethodDelete, "/quizzes/"+uuid.New().String()+"/questions/"+uuid.New().String(), nil), uid)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- GenerateQuestions ---

func TestGenerateQuestions_Success(t *testing.T) {
	uid := uuid.New()
	r := newRouter(&mockQuizSvc{generateJobID: uuid.New()})
	body, _ := json.Marshal(map[string]string{"text": "Generate questions about math"})
	req := withUser(httptest.NewRequest(http.MethodPost, "/quizzes/"+uuid.New().String()+"/generate", bytes.NewReader(body)), uid)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGenerateQuestions_Error(t *testing.T) {
	uid := uuid.New()
	r := newRouter(&mockQuizSvc{generateErr: errors.New("db error")})
	body, _ := json.Marshal(map[string]string{"text": "text"})
	req := withUser(httptest.NewRequest(http.MethodPost, "/quizzes/"+uuid.New().String()+"/generate", bytes.NewReader(body)), uid)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- DeleteCourseSession ---

func TestDeleteCourseSession_Success(t *testing.T) {
	uid := uuid.New()
	r := newRouter(&mockQuizSvc{})
	req := withUser(httptest.NewRequest(http.MethodDelete, "/sessions/"+uuid.New().String(), nil), uid)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestDeleteCourseSession_HasAttempts(t *testing.T) {
	uid := uuid.New()
	r := newRouter(&mockQuizSvc{deleteSessionErr: apperr.ErrSessionHasAttempts})
	req := withUser(httptest.NewRequest(http.MethodDelete, "/sessions/"+uuid.New().String(), nil), uid)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}
