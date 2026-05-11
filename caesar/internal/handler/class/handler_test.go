package class

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

type mockClassService struct {
	listResult    []domain.ClassListItem
	listErr       error
	createID      uuid.UUID
	createErr     error
	classDetail   *domain.ClassDetail
	classErr      error
	updateErr     error
	deleteErr     error
	removeMember  error
	inviteID      uuid.UUID
	inviteErr     error
	acceptClassID uuid.UUID
	acceptErr     error
	invDetail     *domain.InvitationDetail
	invDetailErr  error
}

func (m *mockClassService) GetMyClasses(_ context.Context, _ uuid.UUID, _ domain.ClassMemberRole) ([]domain.ClassListItem, error) {
	return m.listResult, m.listErr
}
func (m *mockClassService) CreateClass(_ context.Context, _ uuid.UUID, _ string) (uuid.UUID, error) {
	return m.createID, m.createErr
}
func (m *mockClassService) GetClass(_ context.Context, _, _ uuid.UUID) (*domain.ClassDetail, error) {
	return m.classDetail, m.classErr
}
func (m *mockClassService) UpdateClass(_ context.Context, _, _ uuid.UUID, _ string) error {
	return m.updateErr
}
func (m *mockClassService) DeleteClass(_ context.Context, _, _ uuid.UUID) error { return m.deleteErr }
func (m *mockClassService) RemoveMember(_ context.Context, _, _, _ uuid.UUID) error {
	return m.removeMember
}
func (m *mockClassService) GetInviteLink(_ context.Context, _, _ uuid.UUID, _ domain.ClassMemberRole) (uuid.UUID, error) {
	return m.inviteID, m.inviteErr
}
func (m *mockClassService) AcceptInvitation(_ context.Context, _, _ uuid.UUID) (uuid.UUID, error) {
	return m.acceptClassID, m.acceptErr
}
func (m *mockClassService) GetInvitationDetail(_ context.Context, _ uuid.UUID) (*domain.InvitationDetail, error) {
	return m.invDetail, m.invDetailErr
}

func newRouter(svc classService) *gin.Engine {
	r := gin.New()
	h := NewHandler(svc)
	classID := ":classId"
	r.GET("/classes", h.GetMyClasses)
	r.POST("/classes", h.CreateClass)
	r.GET("/classes/"+classID, h.GetClass)
	r.PATCH("/classes/"+classID, h.UpdateClass)
	r.DELETE("/classes/"+classID, h.DeleteClass)
	r.DELETE("/classes/"+classID+"/members/:userId", h.RemoveMember)
	r.GET("/classes/"+classID+"/invite", h.GetInviteLink)
	r.POST("/invitations/:invitationId/accept", h.AcceptInvitation)
	r.GET("/invitations/:invitationId", h.GetInvitation)
	return r
}

func withUser(req *http.Request, id uuid.UUID) *http.Request {
	return req.WithContext(middleware.WithUserID(req.Context(), id))
}

func postJSON(r *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// --- GetMyClasses ---

func TestGetMyClasses_NoAuth(t *testing.T) {
	r := newRouter(&mockClassService{})
	req := httptest.NewRequest(http.MethodGet, "/classes?role=teacher", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGetMyClasses_InvalidRole(t *testing.T) {
	r := newRouter(&mockClassService{})
	req := withUser(httptest.NewRequest(http.MethodGet, "/classes?role=admin", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetMyClasses_Success(t *testing.T) {
	classes := []domain.ClassListItem{{Class: domain.Class{ID: uuid.New(), Title: "Math"}}}
	r := newRouter(&mockClassService{listResult: classes})
	req := withUser(httptest.NewRequest(http.MethodGet, "/classes?role=teacher", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	list, _ := resp["classes"].([]any)
	if len(list) != 1 {
		t.Errorf("expected 1 class, got %d", len(list))
	}
}

// --- CreateClass ---

func TestCreateClass_Success(t *testing.T) {
	id := uuid.New()
	r := newRouter(&mockClassService{createID: id})
	body, _ := json.Marshal(map[string]string{"title": "Physics"})
	req := withUser(httptest.NewRequest(http.MethodPost, "/classes", bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCreateClass_EmptyTitle(t *testing.T) {
	r := newRouter(&mockClassService{createErr: apperr.ErrClassEmptyTitle})
	body, _ := json.Marshal(map[string]string{"title": ""})
	req := withUser(httptest.NewRequest(http.MethodPost, "/classes", bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}

// --- GetClass ---

func TestGetClass_InvalidClassID(t *testing.T) {
	r := newRouter(&mockClassService{})
	req := withUser(httptest.NewRequest(http.MethodGet, "/classes/not-uuid", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetClass_Success(t *testing.T) {
	detail := &domain.ClassDetail{Class: domain.Class{ID: uuid.New(), Title: "Math"}}
	r := newRouter(&mockClassService{classDetail: detail})
	req := withUser(httptest.NewRequest(http.MethodGet, "/classes/"+uuid.New().String(), nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- UpdateClass ---

func TestUpdateClass_Success(t *testing.T) {
	r := newRouter(&mockClassService{})
	body, _ := json.Marshal(map[string]string{"title": "New Title"})
	req := withUser(httptest.NewRequest(http.MethodPatch, "/classes/"+uuid.New().String(), bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestUpdateClass_Forbidden(t *testing.T) {
	r := newRouter(&mockClassService{updateErr: apperr.ErrClassForbidden})
	body, _ := json.Marshal(map[string]string{"title": "X"})
	req := withUser(httptest.NewRequest(http.MethodPatch, "/classes/"+uuid.New().String(), bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

// --- DeleteClass ---

func TestDeleteClass_Success(t *testing.T) {
	r := newRouter(&mockClassService{})
	req := withUser(httptest.NewRequest(http.MethodDelete, "/classes/"+uuid.New().String(), nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

// --- RemoveMember ---

func TestRemoveMember_Success(t *testing.T) {
	r := newRouter(&mockClassService{})
	path := "/classes/" + uuid.New().String() + "/members/" + uuid.New().String()
	req := withUser(httptest.NewRequest(http.MethodDelete, path, nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestRemoveMember_CannotRemoveOwner(t *testing.T) {
	r := newRouter(&mockClassService{removeMember: apperr.ErrClassRemoveOwner})
	path := "/classes/" + uuid.New().String() + "/members/" + uuid.New().String()
	req := withUser(httptest.NewRequest(http.MethodDelete, path, nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

// --- GetInviteLink ---

func TestGetInviteLink_InvalidRole(t *testing.T) {
	r := newRouter(&mockClassService{})
	path := "/classes/" + uuid.New().String() + "/invite?role=admin"
	req := withUser(httptest.NewRequest(http.MethodGet, path, nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetInviteLink_Success(t *testing.T) {
	r := newRouter(&mockClassService{inviteID: uuid.New()})
	path := "/classes/" + uuid.New().String() + "/invite?role=student"
	req := withUser(httptest.NewRequest(http.MethodGet, path, nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- AcceptInvitation ---

func TestAcceptInvitation_InvalidUUID(t *testing.T) {
	r := newRouter(&mockClassService{})
	req := withUser(httptest.NewRequest(http.MethodPost, "/invitations/bad-uuid/accept", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAcceptInvitation_AlreadyMember(t *testing.T) {
	r := newRouter(&mockClassService{acceptErr: apperr.ErrAlreadyMember})
	req := withUser(httptest.NewRequest(http.MethodPost, "/invitations/"+uuid.New().String()+"/accept", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestAcceptInvitation_Success(t *testing.T) {
	r := newRouter(&mockClassService{acceptClassID: uuid.New()})
	req := withUser(httptest.NewRequest(http.MethodPost, "/invitations/"+uuid.New().String()+"/accept", nil), uuid.New())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- GetInvitation ---

func TestGetInvitation_NotFound(t *testing.T) {
	r := newRouter(&mockClassService{invDetailErr: apperr.ErrInvitationNotFound})
	req := httptest.NewRequest(http.MethodGet, "/invitations/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetInvitation_Success(t *testing.T) {
	detail := &domain.InvitationDetail{ClassTitle: "Math", Role: domain.ClassMemberRoleStudent}
	r := newRouter(&mockClassService{invDetail: detail})
	req := httptest.NewRequest(http.MethodGet, "/invitations/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["class_title"] != "Math" {
		t.Errorf("unexpected title: %v", resp["class_title"])
	}
}

func TestCreateClass_Error(t *testing.T) {
	r := newRouter(&mockClassService{createErr: errors.New("db error")})
	body, _ := json.Marshal(map[string]string{"title": "X"})
	req := withUser(httptest.NewRequest(http.MethodPost, "/classes", bytes.NewReader(body)), uuid.New())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
