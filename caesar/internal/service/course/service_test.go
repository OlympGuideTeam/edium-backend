package course

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	"caesar/internal/domain"
	"caesar/internal/pkg/apperr"
)

// --- mocks ---

type mockCourseStore struct {
	createID         uuid.UUID
	createErr        error
	byID             *domain.Course
	byIDErr          error
	detail           *domain.CourseDetail
	detailErr        error
	listByClass      []domain.CourseListItem
	listByClassErr   error
	listByStudent    []domain.CourseListItem
	listByStudentErr error
	updateErr        error
	deleteErr        error

	moduleByID      *domain.CourseModule
	moduleByIDErr   error
	modules         []domain.CourseModule
	modulesErr      error
	createModuleID  uuid.UUID
	createModuleErr error
	updateModuleErr error
	deleteModuleErr error
	reorderErr      error

	itemByID       *domain.CourseItem
	itemByIDErr    error
	deleteItemErr  error
	findByObjID    *domain.CourseItem
	findByObjIDErr error

	draftByID      *domain.CourseDraft
	draftByIDErr   error
	deleteDraftErr error
	listDrafts     []domain.CourseDraft

	sheetItems     []domain.CourseSheetItem
	sheetItemsErr  error
	sheetScores    []domain.UserItemScore
	sheetScoresErr error
}

func (m *mockCourseStore) Create(_ context.Context, _, _ uuid.UUID, _ string) (uuid.UUID, error) {
	return m.createID, m.createErr
}
func (m *mockCourseStore) GetByID(_ context.Context, _ uuid.UUID) (*domain.Course, error) {
	return m.byID, m.byIDErr
}
func (m *mockCourseStore) GetDetail(_ context.Context, _ uuid.UUID) (*domain.CourseDetail, error) {
	return m.detail, m.detailErr
}
func (m *mockCourseStore) ListByClassID(_ context.Context, _ uuid.UUID) ([]domain.CourseListItem, error) {
	return m.listByClass, m.listByClassErr
}
func (m *mockCourseStore) ListByStudentID(_ context.Context, _ uuid.UUID) ([]domain.CourseListItem, error) {
	return m.listByStudent, m.listByStudentErr
}
func (m *mockCourseStore) Update(_ context.Context, _ uuid.UUID, _ string) error { return m.updateErr }
func (m *mockCourseStore) Delete(_ context.Context, _ uuid.UUID) error           { return m.deleteErr }

func (m *mockCourseStore) CreateModule(_ context.Context, _ uuid.UUID, _ string) (uuid.UUID, error) {
	return m.createModuleID, m.createModuleErr
}
func (m *mockCourseStore) GetModuleByID(_ context.Context, _ uuid.UUID) (*domain.CourseModule, error) {
	return m.moduleByID, m.moduleByIDErr
}
func (m *mockCourseStore) ListModules(_ context.Context, _ uuid.UUID) ([]domain.CourseModule, error) {
	return m.modules, m.modulesErr
}
func (m *mockCourseStore) UpdateModule(_ context.Context, _ uuid.UUID, _ string) error {
	return m.updateModuleErr
}
func (m *mockCourseStore) DeleteModule(_ context.Context, _ uuid.UUID) error {
	return m.deleteModuleErr
}
func (m *mockCourseStore) ReorderModules(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return m.reorderErr
}

func (m *mockCourseStore) CreateItem(_ context.Context, _, _ uuid.UUID, _ domain.CourseItemType, _ json.RawMessage) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockCourseStore) GetItemByID(_ context.Context, _ uuid.UUID) (*domain.CourseItem, error) {
	return m.itemByID, m.itemByIDErr
}
func (m *mockCourseStore) FindItemByObjectID(_ context.Context, _ uuid.UUID) (*domain.CourseItem, error) {
	return m.findByObjID, m.findByObjIDErr
}
func (m *mockCourseStore) DeleteItem(_ context.Context, _ uuid.UUID) error { return m.deleteItemErr }
func (m *mockCourseStore) ListItemsByModuleIDs(_ context.Context, _ []uuid.UUID, _ uuid.UUID) ([]domain.CourseModuleItem, error) {
	return nil, nil
}
func (m *mockCourseStore) UpsertProgress(_ context.Context, _, _, _ uuid.UUID) error { return nil }
func (m *mockCourseStore) UpdateProgressScore(_ context.Context, _, _ uuid.UUID, _ float64) error {
	return nil
}
func (m *mockCourseStore) UpsertCourseDraft(_ context.Context, _, _ uuid.UUID, _ string, _ json.RawMessage) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockCourseStore) GetDraftByID(_ context.Context, _ uuid.UUID) (*domain.CourseDraft, error) {
	return m.draftByID, m.draftByIDErr
}
func (m *mockCourseStore) DeleteDraft(_ context.Context, _ uuid.UUID) error { return m.deleteDraftErr }
func (m *mockCourseStore) ListDraftsByCourseID(_ context.Context, _ uuid.UUID) ([]domain.CourseDraft, error) {
	return m.listDrafts, nil
}
func (m *mockCourseStore) GetSheetItems(_ context.Context, _ uuid.UUID) ([]domain.CourseSheetItem, error) {
	return m.sheetItems, m.sheetItemsErr
}
func (m *mockCourseStore) GetSheetScores(_ context.Context, _ uuid.UUID) ([]domain.UserItemScore, error) {
	return m.sheetScores, m.sheetScoresErr
}

type mockClassAccessor struct {
	classItem  *domain.ClassListItem
	classErr   error
	role       domain.ClassMemberRole
	isMember   bool
	roleErr    error
	members    []domain.ClassMember
	membersErr error
}

func (m *mockClassAccessor) GetByID(_ context.Context, _ uuid.UUID) (*domain.ClassListItem, error) {
	return m.classItem, m.classErr
}
func (m *mockClassAccessor) GetMemberRole(_ context.Context, _, _ uuid.UUID) (domain.ClassMemberRole, bool, error) {
	return m.role, m.isMember, m.roleErr
}
func (m *mockClassAccessor) GetMembersForDetail(_ context.Context, _ uuid.UUID) ([]domain.ClassMember, error) {
	return m.members, m.membersErr
}

type mockTaskScheduler struct{ err error }

func (m *mockTaskScheduler) Schedule(_ context.Context, _ domain.TaskType, _ []byte) error {
	return m.err
}

type mockTx struct{ err error }

func (m *mockTx) WithTx(ctx context.Context, fn func(context.Context) error) error {
	if m.err != nil {
		return m.err
	}
	return fn(ctx)
}

func ownerID() uuid.UUID { return uuid.MustParse("00000000-0000-0000-0000-000000000001") }
func otherID() uuid.UUID { return uuid.MustParse("00000000-0000-0000-0000-000000000002") }

func courseWith(classID, ownerID uuid.UUID) *domain.Course {
	return &domain.Course{ID: uuid.New(), ClassID: classID, OwnerID: ownerID}
}

func classItem(ownerID uuid.UUID) *domain.ClassListItem {
	return &domain.ClassListItem{Class: domain.Class{ID: uuid.New(), OwnerID: ownerID}}
}

func newSvc(cs *mockCourseStore, ca *mockClassAccessor, task *mockTaskScheduler, tx *mockTx) *Service {
	return NewService(cs, ca, task, tx)
}

// --- CreateCourse ---

func TestCreateCourse_EmptyTitle(t *testing.T) {
	svc := newSvc(&mockCourseStore{}, &mockClassAccessor{}, &mockTaskScheduler{}, &mockTx{})
	_, err := svc.CreateCourse(context.Background(), uuid.New(), ownerID(), "   ")
	if !errors.Is(err, apperr.ErrCourseEmptyTitle) {
		t.Fatalf("expected ErrCourseEmptyTitle, got %v", err)
	}
}

func TestCreateCourse_NotMember(t *testing.T) {
	ca := &mockClassAccessor{isMember: false}
	svc := newSvc(&mockCourseStore{}, ca, &mockTaskScheduler{}, &mockTx{})
	_, err := svc.CreateCourse(context.Background(), uuid.New(), ownerID(), "Math")
	if !errors.Is(err, apperr.ErrCourseNotMember) {
		t.Fatalf("expected ErrCourseNotMember, got %v", err)
	}
}

func TestCreateCourse_StudentForbidden(t *testing.T) {
	ca := &mockClassAccessor{role: domain.ClassMemberRoleStudent, isMember: true}
	svc := newSvc(&mockCourseStore{}, ca, &mockTaskScheduler{}, &mockTx{})
	_, err := svc.CreateCourse(context.Background(), uuid.New(), ownerID(), "Math")
	if !errors.Is(err, apperr.ErrCourseForbidden) {
		t.Fatalf("expected ErrCourseForbidden, got %v", err)
	}
}

func TestCreateCourse_Success(t *testing.T) {
	id := uuid.New()
	cs := &mockCourseStore{createID: id}
	ca := &mockClassAccessor{role: domain.ClassMemberRoleTeacher, isMember: true}
	svc := newSvc(cs, ca, &mockTaskScheduler{}, &mockTx{})
	got, err := svc.CreateCourse(context.Background(), uuid.New(), ownerID(), "Math")
	if err != nil || got != id {
		t.Fatalf("unexpected: err=%v id=%v", err, got)
	}
}

// --- GetCourse ---

func TestGetCourse_NotFound(t *testing.T) {
	svc := newSvc(&mockCourseStore{}, &mockClassAccessor{}, &mockTaskScheduler{}, &mockTx{})
	_, err := svc.GetCourse(context.Background(), uuid.New(), ownerID())
	if !errors.Is(err, apperr.ErrCourseNotFound) {
		t.Fatalf("expected ErrCourseNotFound, got %v", err)
	}
}

func TestGetCourse_NotMember(t *testing.T) {
	classID := uuid.New()
	cs := &mockCourseStore{detail: &domain.CourseDetail{Course: domain.Course{ClassID: classID}}}
	ca := &mockClassAccessor{isMember: false}
	svc := newSvc(cs, ca, &mockTaskScheduler{}, &mockTx{})
	_, err := svc.GetCourse(context.Background(), uuid.New(), ownerID())
	if !errors.Is(err, apperr.ErrCourseNotMember) {
		t.Fatalf("expected ErrCourseNotMember, got %v", err)
	}
}

func TestGetCourse_Success_Teacher(t *testing.T) {
	classID := uuid.New()
	cs := &mockCourseStore{detail: &domain.CourseDetail{Course: domain.Course{ClassID: classID}}}
	ca := &mockClassAccessor{role: domain.ClassMemberRoleTeacher, isMember: true}
	svc := newSvc(cs, ca, &mockTaskScheduler{}, &mockTx{})
	detail, err := svc.GetCourse(context.Background(), uuid.New(), ownerID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !detail.IsTeacher {
		t.Error("expected IsTeacher=true")
	}
}

func TestGetCourse_Success_Student(t *testing.T) {
	classID := uuid.New()
	cs := &mockCourseStore{detail: &domain.CourseDetail{Course: domain.Course{ClassID: classID}}}
	ca := &mockClassAccessor{role: domain.ClassMemberRoleStudent, isMember: true}
	svc := newSvc(cs, ca, &mockTaskScheduler{}, &mockTx{})
	detail, err := svc.GetCourse(context.Background(), uuid.New(), ownerID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.IsTeacher {
		t.Error("expected IsTeacher=false")
	}
}

// --- UpdateCourse ---

func TestUpdateCourse_EmptyTitle(t *testing.T) {
	svc := newSvc(&mockCourseStore{}, &mockClassAccessor{}, &mockTaskScheduler{}, &mockTx{})
	if err := svc.UpdateCourse(context.Background(), uuid.New(), ownerID(), " "); !errors.Is(err, apperr.ErrCourseEmptyTitle) {
		t.Fatalf("expected ErrCourseEmptyTitle, got %v", err)
	}
}

func TestUpdateCourse_Forbidden_Student(t *testing.T) {
	classID := uuid.New()
	cs := &mockCourseStore{byID: courseWith(classID, otherID())}
	ca := &mockClassAccessor{role: domain.ClassMemberRoleStudent, isMember: true}
	svc := newSvc(cs, ca, &mockTaskScheduler{}, &mockTx{})
	if err := svc.UpdateCourse(context.Background(), uuid.New(), ownerID(), "New"); !errors.Is(err, apperr.ErrCourseForbidden) {
		t.Fatalf("expected ErrCourseForbidden, got %v", err)
	}
}

func TestUpdateCourse_Success(t *testing.T) {
	classID := uuid.New()
	oid := ownerID()
	cs := &mockCourseStore{byID: courseWith(classID, oid)}
	ca := &mockClassAccessor{role: domain.ClassMemberRoleTeacher, isMember: true}
	svc := newSvc(cs, ca, &mockTaskScheduler{}, &mockTx{})
	if err := svc.UpdateCourse(context.Background(), uuid.New(), oid, "New Title"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- DeleteCourse ---

func TestDeleteCourse_NotFound(t *testing.T) {
	svc := newSvc(&mockCourseStore{}, &mockClassAccessor{}, &mockTaskScheduler{}, &mockTx{})
	if err := svc.DeleteCourse(context.Background(), uuid.New(), ownerID()); !errors.Is(err, apperr.ErrCourseNotFound) {
		t.Fatalf("expected ErrCourseNotFound, got %v", err)
	}
}

func TestDeleteCourse_Owner_Success(t *testing.T) {
	oid := ownerID()
	cs := &mockCourseStore{byID: courseWith(uuid.New(), oid)}
	svc := newSvc(cs, &mockClassAccessor{}, &mockTaskScheduler{}, &mockTx{})
	if err := svc.DeleteCourse(context.Background(), uuid.New(), oid); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteCourse_ClassOwner_Success(t *testing.T) {
	classID := uuid.New()
	classOwnerID := ownerID()
	courseOwnerID := otherID()
	cs := &mockCourseStore{byID: courseWith(classID, courseOwnerID)}
	ca := &mockClassAccessor{classItem: classItem(classOwnerID)}
	svc := newSvc(cs, ca, &mockTaskScheduler{}, &mockTx{})
	if err := svc.DeleteCourse(context.Background(), uuid.New(), classOwnerID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteCourse_Forbidden(t *testing.T) {
	classID := uuid.New()
	cs := &mockCourseStore{byID: courseWith(classID, otherID())}
	ca := &mockClassAccessor{classItem: classItem(otherID())}
	svc := newSvc(cs, ca, &mockTaskScheduler{}, &mockTx{})
	if err := svc.DeleteCourse(context.Background(), uuid.New(), ownerID()); !errors.Is(err, apperr.ErrCourseForbidden) {
		t.Fatalf("expected ErrCourseForbidden, got %v", err)
	}
}

// --- CreateModule ---

func TestCreateModule_EmptyTitle(t *testing.T) {
	svc := newSvc(&mockCourseStore{}, &mockClassAccessor{}, &mockTaskScheduler{}, &mockTx{})
	_, err := svc.CreateModule(context.Background(), uuid.New(), ownerID(), "")
	if !errors.Is(err, apperr.ErrCourseEmptyTitle) {
		t.Fatalf("expected ErrCourseEmptyTitle, got %v", err)
	}
}

func TestCreateModule_Success(t *testing.T) {
	oid := ownerID()
	modID := uuid.New()
	cs := &mockCourseStore{byID: courseWith(uuid.New(), oid), createModuleID: modID}
	svc := newSvc(cs, &mockClassAccessor{}, &mockTaskScheduler{}, &mockTx{})
	got, err := svc.CreateModule(context.Background(), uuid.New(), oid, "Module 1")
	if err != nil || got != modID {
		t.Fatalf("unexpected: err=%v id=%v", err, got)
	}
}

// --- DeleteModule ---

func TestDeleteModule_NotFound(t *testing.T) {
	svc := newSvc(&mockCourseStore{}, &mockClassAccessor{}, &mockTaskScheduler{}, &mockTx{})
	if err := svc.DeleteModule(context.Background(), uuid.New(), ownerID()); !errors.Is(err, apperr.ErrModuleNotFound) {
		t.Fatalf("expected ErrModuleNotFound, got %v", err)
	}
}

func TestDeleteModule_Success(t *testing.T) {
	oid := ownerID()
	courseID := uuid.New()
	cs := &mockCourseStore{
		moduleByID: &domain.CourseModule{ID: uuid.New(), CourseID: courseID},
		byID:       courseWith(uuid.New(), oid),
	}
	svc := newSvc(cs, &mockClassAccessor{}, &mockTaskScheduler{}, &mockTx{})
	if err := svc.DeleteModule(context.Background(), uuid.New(), oid); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- GetCourseSheet ---

func TestGetCourseSheet_CourseNotFound(t *testing.T) {
	svc := newSvc(&mockCourseStore{}, &mockClassAccessor{}, &mockTaskScheduler{}, &mockTx{})
	_, err := svc.GetCourseSheet(context.Background(), uuid.New(), ownerID())
	if !errors.Is(err, apperr.ErrCourseNotFound) {
		t.Fatalf("expected ErrCourseNotFound, got %v", err)
	}
}

func TestGetCourseSheet_NotMember(t *testing.T) {
	classID := uuid.New()
	cs := &mockCourseStore{byID: courseWith(classID, ownerID())}
	ca := &mockClassAccessor{isMember: false}
	svc := newSvc(cs, ca, &mockTaskScheduler{}, &mockTx{})
	_, err := svc.GetCourseSheet(context.Background(), uuid.New(), ownerID())
	if !errors.Is(err, apperr.ErrCourseNotMember) {
		t.Fatalf("expected ErrCourseNotMember, got %v", err)
	}
}

func TestGetCourseSheet_Success_Empty(t *testing.T) {
	classID := uuid.New()
	cs := &mockCourseStore{byID: courseWith(classID, ownerID())}
	ca := &mockClassAccessor{isMember: true, role: domain.ClassMemberRoleTeacher}
	svc := newSvc(cs, ca, &mockTaskScheduler{}, &mockTx{})
	sheet, err := svc.GetCourseSheet(context.Background(), uuid.New(), ownerID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sheet.Students) != 0 || len(sheet.Items) != 0 {
		t.Errorf("expected empty sheet, got items=%d students=%d", len(sheet.Items), len(sheet.Students))
	}
}

func TestGetCourseSheet_Success_WithStudentsAndScores(t *testing.T) {
	classID := uuid.New()
	itemID := uuid.New()
	studentID := uuid.New()
	score := 8.5

	cs := &mockCourseStore{
		byID:       courseWith(classID, ownerID()),
		sheetItems: []domain.CourseSheetItem{{ID: itemID, Title: "Quiz 1"}},
		sheetScores: []domain.UserItemScore{
			{UserID: studentID, ItemID: itemID, Score: &score},
		},
	}
	ca := &mockClassAccessor{
		isMember: true,
		role:     domain.ClassMemberRoleTeacher,
		members: []domain.ClassMember{
			{UserID: studentID, Role: domain.ClassMemberRoleStudent, Name: "Петя", Surname: "Иванов"},
		},
	}
	svc := newSvc(cs, ca, &mockTaskScheduler{}, &mockTx{})
	sheet, err := svc.GetCourseSheet(context.Background(), uuid.New(), ownerID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sheet.Students) != 1 {
		t.Fatalf("expected 1 student row, got %d", len(sheet.Students))
	}
	if len(sheet.Students[0].Scores) != 1 {
		t.Fatalf("expected 1 score, got %d", len(sheet.Students[0].Scores))
	}
	if *sheet.Students[0].Scores[0].Score != score {
		t.Errorf("unexpected score: %v", sheet.Students[0].Scores[0].Score)
	}
}
