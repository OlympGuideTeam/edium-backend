package course

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

type mockCourseService struct {
	createID        uuid.UUID
	createErr       error
	courseDetail    *domain.CourseDetail
	courseErr       error
	classCourses    []domain.CourseListItem
	classCoursesErr error
	myCourses       []domain.CourseListItem
	myCoursesErr    error
	updateErr       error
	deleteErr       error
	module          *domain.CourseModule
	moduleErr       error
	roster          []domain.ClassMember
	rosterErr       error
	createModuleID  uuid.UUID
	createModErr    error
	updateModErr    error
	deleteModErr    error
	reorderErr      error
	deleteItemErr   error
	deleteDraftErr  error
	sheet           *domain.CourseSheet
	sheetErr        error
}

func (m *mockCourseService) CreateCourse(_ context.Context, _, _ uuid.UUID, _ string) (uuid.UUID, error) {
	return m.createID, m.createErr
}
func (m *mockCourseService) GetCourse(_ context.Context, _, _ uuid.UUID) (*domain.CourseDetail, error) {
	return m.courseDetail, m.courseErr
}
func (m *mockCourseService) GetClassCourses(_ context.Context, _, _ uuid.UUID) ([]domain.CourseListItem, error) {
	return m.classCourses, m.classCoursesErr
}
func (m *mockCourseService) GetMyCourses(_ context.Context, _ uuid.UUID) ([]domain.CourseListItem, error) {
	return m.myCourses, m.myCoursesErr
}
func (m *mockCourseService) UpdateCourse(_ context.Context, _, _ uuid.UUID, _ string) error {
	return m.updateErr
}
func (m *mockCourseService) DeleteCourse(_ context.Context, _, _ uuid.UUID) error { return m.deleteErr }
func (m *mockCourseService) GetModule(_ context.Context, _, _ uuid.UUID) (*domain.CourseModule, error) {
	return m.module, m.moduleErr
}
func (m *mockCourseService) GetModuleRoster(_ context.Context, _, _ uuid.UUID) ([]domain.ClassMember, error) {
	return m.roster, m.rosterErr
}
func (m *mockCourseService) CreateModule(_ context.Context, _, _ uuid.UUID, _ string) (uuid.UUID, error) {
	return m.createModuleID, m.createModErr
}
func (m *mockCourseService) UpdateModule(_ context.Context, _, _ uuid.UUID, _ string) error {
	return m.updateModErr
}
func (m *mockCourseService) DeleteModule(_ context.Context, _, _ uuid.UUID) error {
	return m.deleteModErr
}
func (m *mockCourseService) ReorderModules(_ context.Context, _, _ uuid.UUID, _ []uuid.UUID) error {
	return m.reorderErr
}
func (m *mockCourseService) DeleteCourseItem(_ context.Context, _, _ uuid.UUID) error {
	return m.deleteItemErr
}
func (m *mockCourseService) DeleteCourseDraft(_ context.Context, _, _ uuid.UUID) error {
	return m.deleteDraftErr
}
func (m *mockCourseService) GetCourseSheet(_ context.Context, _, _ uuid.UUID) (*domain.CourseSheet, error) {
	return m.sheet, m.sheetErr
}

func newRouter(svc courseService) *gin.Engine {
	r := gin.New()
	h := NewHandler(svc)
	r.POST("/courses", h.CreateCourse)
	r.GET("/courses/:courseId", h.GetCourse)
	r.GET("/courses/:courseId/sheet", h.GetCourseSheet)
	r.PATCH("/courses/:courseId", h.UpdateCourse)
	r.DELETE("/courses/:courseId", h.DeleteCourse)
	r.GET("/classes/:classId/courses", h.GetClassCourses)
	r.GET("/courses/me", h.GetMyCourses)
	r.GET("/modules/:moduleId", h.GetModule)
	r.GET("/modules/:moduleId/roster", h.GetModuleRoster)
	r.POST("/courses/:courseId/modules", h.CreateModule)
	r.PATCH("/modules/:moduleId", h.UpdateModule)
	r.DELETE("/modules/:moduleId", h.DeleteModule)
	r.PUT("/courses/:courseId/modules/reorder", h.ReorderModules)
	r.DELETE("/items/:itemId", h.DeleteCourseItem)
	r.DELETE("/drafts/:draftId", h.DeleteCourseDraft)
	return r
}

func withUser(req *http.Request, id uuid.UUID) *http.Request {
	return req.WithContext(middleware.WithUserID(req.Context(), id))
}

// --- CreateCourse ---

func TestCreateCourse_NoAuth(t *testing.T) {
	r := newRouter(&mockCourseService{})
	body, _ := json.Marshal(map[string]string{"title": "Math", "class_id": uuid.New().String()})
	req := httptest.NewRequest(http.MethodPost, "/courses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCreateCourse_BadClassID(t *testing.T) {
	r := newRouter(&mockCourseService{})
	body, _ := json.Marshal(map[string]string{"title": "Math", "class_id": "bad-uuid"})
	req := withUser(httptest.NewRequest(http.MethodPost, "/courses", bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateCourse_Success(t *testing.T) {
	id := uuid.New()
	r := newRouter(&mockCourseService{createID: id})
	body, _ := json.Marshal(map[string]string{"title": "Math", "class_id": uuid.New().String()})
	req := withUser(httptest.NewRequest(http.MethodPost, "/courses", bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- GetCourse ---

func TestGetCourse_InvalidID(t *testing.T) {
	r := newRouter(&mockCourseService{})
	req := withUser(httptest.NewRequest(http.MethodGet, "/courses/bad-uuid", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetCourse_NotFound(t *testing.T) {
	r := newRouter(&mockCourseService{courseErr: apperr.ErrCourseNotFound})
	req := withUser(httptest.NewRequest(http.MethodGet, "/courses/"+uuid.New().String(), nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetCourse_Success(t *testing.T) {
	detail := &domain.CourseDetail{Course: domain.Course{ID: uuid.New(), Title: "Math"}}
	r := newRouter(&mockCourseService{courseDetail: detail})
	req := withUser(httptest.NewRequest(http.MethodGet, "/courses/"+uuid.New().String(), nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- GetClassCourses ---

func TestGetClassCourses_Success(t *testing.T) {
	courses := []domain.CourseListItem{{Course: domain.Course{ID: uuid.New(), Title: "Math"}}}
	r := newRouter(&mockCourseService{classCourses: courses})
	req := withUser(httptest.NewRequest(http.MethodGet, "/classes/"+uuid.New().String()+"/courses", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- GetMyCourses ---

func TestGetMyCourses_Success(t *testing.T) {
	r := newRouter(&mockCourseService{myCourses: []domain.CourseListItem{}})
	req := withUser(httptest.NewRequest(http.MethodGet, "/courses/me", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- UpdateCourse ---

func TestUpdateCourse_Success(t *testing.T) {
	r := newRouter(&mockCourseService{})
	body, _ := json.Marshal(map[string]string{"title": "New"})
	req := withUser(httptest.NewRequest(http.MethodPatch, "/courses/"+uuid.New().String(), bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestUpdateCourse_Forbidden(t *testing.T) {
	r := newRouter(&mockCourseService{updateErr: apperr.ErrCourseForbidden})
	body, _ := json.Marshal(map[string]string{"title": "New"})
	req := withUser(httptest.NewRequest(http.MethodPatch, "/courses/"+uuid.New().String(), bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

// --- DeleteCourse ---

func TestDeleteCourse_Success(t *testing.T) {
	r := newRouter(&mockCourseService{})
	req := withUser(httptest.NewRequest(http.MethodDelete, "/courses/"+uuid.New().String(), nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

// --- GetCourseSheet ---

func TestGetCourseSheet_Success(t *testing.T) {
	sheet := &domain.CourseSheet{Items: []domain.CourseSheetItem{}, Students: []domain.SheetRow{}}
	r := newRouter(&mockCourseService{sheet: sheet})
	req := withUser(httptest.NewRequest(http.MethodGet, "/courses/"+uuid.New().String()+"/sheet", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetCourseSheet_Error(t *testing.T) {
	r := newRouter(&mockCourseService{sheetErr: errors.New("db error")})
	req := withUser(httptest.NewRequest(http.MethodGet, "/courses/"+uuid.New().String()+"/sheet", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- Module handlers ---

func TestGetModule_InvalidID(t *testing.T) {
	r := newRouter(&mockCourseService{})
	req := withUser(httptest.NewRequest(http.MethodGet, "/modules/bad-uuid", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetModule_Success(t *testing.T) {
	mod := &domain.CourseModule{ID: uuid.New(), Title: "Module 1"}
	r := newRouter(&mockCourseService{module: mod})
	req := withUser(httptest.NewRequest(http.MethodGet, "/modules/"+uuid.New().String(), nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetModuleRoster_Success(t *testing.T) {
	members := []domain.ClassMember{{UserID: uuid.New(), Name: "Ivan", Role: domain.ClassMemberRoleStudent}}
	r := newRouter(&mockCourseService{roster: members})
	req := withUser(httptest.NewRequest(http.MethodGet, "/modules/"+uuid.New().String()+"/roster", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCreateModule_Success(t *testing.T) {
	id := uuid.New()
	r := newRouter(&mockCourseService{createModuleID: id})
	body, _ := json.Marshal(map[string]string{"title": "Module 1"})
	req := withUser(httptest.NewRequest(http.MethodPost, "/courses/"+uuid.New().String()+"/modules", bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestUpdateModule_Success(t *testing.T) {
	r := newRouter(&mockCourseService{})
	body, _ := json.Marshal(map[string]string{"title": "Updated"})
	req := withUser(httptest.NewRequest(http.MethodPatch, "/modules/"+uuid.New().String(), bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestDeleteModule_Success(t *testing.T) {
	r := newRouter(&mockCourseService{})
	req := withUser(httptest.NewRequest(http.MethodDelete, "/modules/"+uuid.New().String(), nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestReorderModules_Success(t *testing.T) {
	r := newRouter(&mockCourseService{})
	body, _ := json.Marshal(map[string][]string{"module_ids": {uuid.New().String(), uuid.New().String()}})
	req := withUser(httptest.NewRequest(http.MethodPut, "/courses/"+uuid.New().String()+"/modules/reorder", bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestReorderModules_InvalidModuleID(t *testing.T) {
	r := newRouter(&mockCourseService{})
	body, _ := json.Marshal(map[string][]string{"module_ids": {"bad-uuid"}})
	req := withUser(httptest.NewRequest(http.MethodPut, "/courses/"+uuid.New().String()+"/modules/reorder", bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- Item / Draft ---

func TestDeleteCourseItem_Success(t *testing.T) {
	r := newRouter(&mockCourseService{})
	req := withUser(httptest.NewRequest(http.MethodDelete, "/items/"+uuid.New().String(), nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestDeleteCourseDraft_Success(t *testing.T) {
	r := newRouter(&mockCourseService{})
	req := withUser(httptest.NewRequest(http.MethodDelete, "/drafts/"+uuid.New().String(), nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestDeleteCourseDraft_InvalidID(t *testing.T) {
	r := newRouter(&mockCourseService{})
	req := withUser(httptest.NewRequest(http.MethodDelete, "/drafts/bad-uuid", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDeleteCourseItem_Error(t *testing.T) {
	r := newRouter(&mockCourseService{deleteItemErr: errors.New("db error")})
	req := withUser(httptest.NewRequest(http.MethodDelete, "/items/"+uuid.New().String(), nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
