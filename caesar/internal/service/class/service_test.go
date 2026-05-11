package class

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"caesar/internal/domain"
	"caesar/internal/pkg/apperr"
)

// --- mocks ---

type mockClassStore struct {
	createID      uuid.UUID
	createErr     error
	getResult     *domain.ClassListItem
	getErr        error
	listResult    []domain.ClassListItem
	listErr       error
	updateErr     error
	deleteErr     error
	members       []domain.ClassMember
	membersErr    error
	addMemberErr  error
	removeMember  error
	isMember      bool
	isMemberErr   error
	invitationID  uuid.UUID
	upsertInvErr  error
	invitation    *domain.ClassInvitation
	invErr        error
	invDetail     *domain.InvitationDetail
	invDetailErr  error
}

func (m *mockClassStore) Create(_ context.Context, _ uuid.UUID, _ string) (uuid.UUID, error) {
	return m.createID, m.createErr
}
func (m *mockClassStore) GetByID(_ context.Context, _ uuid.UUID) (*domain.ClassListItem, error) {
	return m.getResult, m.getErr
}
func (m *mockClassStore) ListByUserID(_ context.Context, _ uuid.UUID, _ domain.ClassMemberRole) ([]domain.ClassListItem, error) {
	return m.listResult, m.listErr
}
func (m *mockClassStore) Update(_ context.Context, _ uuid.UUID, _ string) error { return m.updateErr }
func (m *mockClassStore) Delete(_ context.Context, _ uuid.UUID) error           { return m.deleteErr }
func (m *mockClassStore) AddMember(_ context.Context, _, _ uuid.UUID, _ domain.ClassMemberRole) error {
	return m.addMemberErr
}
func (m *mockClassStore) GetMembersForDetail(_ context.Context, _ uuid.UUID) ([]domain.ClassMember, error) {
	return m.members, m.membersErr
}
func (m *mockClassStore) RemoveMember(_ context.Context, _, _ uuid.UUID) error { return m.removeMember }
func (m *mockClassStore) IsMember(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return m.isMember, m.isMemberErr
}
func (m *mockClassStore) UpsertInvitation(_ context.Context, _ uuid.UUID, _ domain.ClassMemberRole) (uuid.UUID, error) {
	return m.invitationID, m.upsertInvErr
}
func (m *mockClassStore) GetInvitation(_ context.Context, _ uuid.UUID) (*domain.ClassInvitation, error) {
	return m.invitation, m.invErr
}
func (m *mockClassStore) GetInvitationWithClass(_ context.Context, _ uuid.UUID) (*domain.InvitationDetail, error) {
	return m.invDetail, m.invDetailErr
}

type mockCourseAccessor struct {
	items []domain.CourseListItem
	err   error
}

func (m *mockCourseAccessor) ListByClassID(_ context.Context, _ uuid.UUID) ([]domain.CourseListItem, error) {
	return m.items, m.err
}

func ownerID() uuid.UUID { return uuid.MustParse("00000000-0000-0000-0000-000000000001") }
func otherID() uuid.UUID { return uuid.MustParse("00000000-0000-0000-0000-000000000002") }

func classItem(ownerID uuid.UUID) *domain.ClassListItem {
	return &domain.ClassListItem{Class: domain.Class{ID: uuid.New(), OwnerID: ownerID, Title: "Математика"}}
}

func newSvc(cs *mockClassStore, ca *mockCourseAccessor) *Service {
	return NewService(cs, ca)
}

// --- GetMyClasses ---

func TestGetMyClasses_Success(t *testing.T) {
	cs := &mockClassStore{listResult: []domain.ClassListItem{*classItem(ownerID())}}
	svc := newSvc(cs, &mockCourseAccessor{})
	result, err := svc.GetMyClasses(context.Background(), ownerID(), domain.ClassMemberRoleTeacher)
	if err != nil || len(result) != 1 {
		t.Fatalf("unexpected: err=%v len=%d", err, len(result))
	}
}

func TestGetMyClasses_Error(t *testing.T) {
	cs := &mockClassStore{listErr: errors.New("db error")}
	svc := newSvc(cs, &mockCourseAccessor{})
	_, err := svc.GetMyClasses(context.Background(), ownerID(), domain.ClassMemberRoleTeacher)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- CreateClass ---

func TestCreateClass_EmptyTitle(t *testing.T) {
	svc := newSvc(&mockClassStore{}, &mockCourseAccessor{})
	_, err := svc.CreateClass(context.Background(), ownerID(), "")
	if !errors.Is(err, apperr.ErrClassEmptyTitle) {
		t.Fatalf("expected ErrClassEmptyTitle, got %v", err)
	}
}

func TestCreateClass_StoreError(t *testing.T) {
	cs := &mockClassStore{createErr: errors.New("db error")}
	svc := newSvc(cs, &mockCourseAccessor{})
	_, err := svc.CreateClass(context.Background(), ownerID(), "Физика")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateClass_Success(t *testing.T) {
	id := uuid.New()
	cs := &mockClassStore{createID: id}
	svc := newSvc(cs, &mockCourseAccessor{})
	got, err := svc.CreateClass(context.Background(), ownerID(), "Физика")
	if err != nil || got != id {
		t.Fatalf("unexpected: err=%v id=%v", err, got)
	}
}

// --- GetClass ---

func TestGetClass_NotFound(t *testing.T) {
	svc := newSvc(&mockClassStore{}, &mockCourseAccessor{})
	_, err := svc.GetClass(context.Background(), uuid.New(), ownerID())
	if !errors.Is(err, apperr.ErrClassNotFound) {
		t.Fatalf("expected ErrClassNotFound, got %v", err)
	}
}

func TestGetClass_Success_Owner(t *testing.T) {
	oid := ownerID()
	cs := &mockClassStore{getResult: classItem(oid)}
	svc := newSvc(cs, &mockCourseAccessor{})
	detail, err := svc.GetClass(context.Background(), uuid.New(), oid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !detail.IsOwner {
		t.Error("expected IsOwner=true")
	}
}

func TestGetClass_Success_TeacherMember(t *testing.T) {
	oid := ownerID()
	tid := otherID()
	cs := &mockClassStore{
		getResult: classItem(oid),
		members: []domain.ClassMember{
			{UserID: tid, Role: domain.ClassMemberRoleTeacher, Name: "Anna"},
		},
	}
	svc := newSvc(cs, &mockCourseAccessor{})
	detail, err := svc.GetClass(context.Background(), uuid.New(), tid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(detail.Teachers) != 1 {
		t.Errorf("expected 1 teacher, got %d", len(detail.Teachers))
	}
}

// --- UpdateClass ---

func TestUpdateClass_EmptyTitle(t *testing.T) {
	svc := newSvc(&mockClassStore{}, &mockCourseAccessor{})
	err := svc.UpdateClass(context.Background(), uuid.New(), ownerID(), "")
	if !errors.Is(err, apperr.ErrClassEmptyTitle) {
		t.Fatalf("expected ErrClassEmptyTitle, got %v", err)
	}
}

func TestUpdateClass_NotFound(t *testing.T) {
	svc := newSvc(&mockClassStore{}, &mockCourseAccessor{})
	if err := svc.UpdateClass(context.Background(), uuid.New(), ownerID(), "New"); !errors.Is(err, apperr.ErrClassNotFound) {
		t.Fatalf("expected ErrClassNotFound, got %v", err)
	}
}

func TestUpdateClass_Forbidden(t *testing.T) {
	cs := &mockClassStore{getResult: classItem(ownerID())}
	svc := newSvc(cs, &mockCourseAccessor{})
	err := svc.UpdateClass(context.Background(), uuid.New(), otherID(), "New")
	if !errors.Is(err, apperr.ErrClassForbidden) {
		t.Fatalf("expected ErrClassForbidden, got %v", err)
	}
}

func TestUpdateClass_Success(t *testing.T) {
	oid := ownerID()
	cs := &mockClassStore{getResult: classItem(oid)}
	svc := newSvc(cs, &mockCourseAccessor{})
	if err := svc.UpdateClass(context.Background(), uuid.New(), oid, "New Title"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- DeleteClass ---

func TestDeleteClass_Forbidden(t *testing.T) {
	cs := &mockClassStore{getResult: classItem(ownerID())}
	svc := newSvc(cs, &mockCourseAccessor{})
	if err := svc.DeleteClass(context.Background(), uuid.New(), otherID()); !errors.Is(err, apperr.ErrClassForbidden) {
		t.Fatalf("expected ErrClassForbidden, got %v", err)
	}
}

func TestDeleteClass_Success(t *testing.T) {
	oid := ownerID()
	cs := &mockClassStore{getResult: classItem(oid)}
	svc := newSvc(cs, &mockCourseAccessor{})
	if err := svc.DeleteClass(context.Background(), uuid.New(), oid); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- GetInviteLink ---

func TestGetInviteLink_Forbidden(t *testing.T) {
	cs := &mockClassStore{getResult: classItem(ownerID())}
	svc := newSvc(cs, &mockCourseAccessor{})
	_, err := svc.GetInviteLink(context.Background(), uuid.New(), otherID(), domain.ClassMemberRoleStudent)
	if !errors.Is(err, apperr.ErrClassForbidden) {
		t.Fatalf("expected ErrClassForbidden, got %v", err)
	}
}

func TestGetInviteLink_Success(t *testing.T) {
	oid := ownerID()
	invID := uuid.New()
	cs := &mockClassStore{getResult: classItem(oid), invitationID: invID}
	svc := newSvc(cs, &mockCourseAccessor{})
	got, err := svc.GetInviteLink(context.Background(), uuid.New(), oid, domain.ClassMemberRoleStudent)
	if err != nil || got != invID {
		t.Fatalf("unexpected: err=%v id=%v", err, got)
	}
}

// --- AcceptInvitation ---

func TestAcceptInvitation_NotFound(t *testing.T) {
	svc := newSvc(&mockClassStore{}, &mockCourseAccessor{})
	_, err := svc.AcceptInvitation(context.Background(), uuid.New(), otherID())
	if !errors.Is(err, apperr.ErrInvitationNotFound) {
		t.Fatalf("expected ErrInvitationNotFound, got %v", err)
	}
}

func TestAcceptInvitation_OwnerCannotJoin(t *testing.T) {
	oid := ownerID()
	cs := &mockClassStore{
		invitation: &domain.ClassInvitation{ClassID: uuid.New(), Role: domain.ClassMemberRoleStudent},
		getResult:  classItem(oid),
	}
	svc := newSvc(cs, &mockCourseAccessor{})
	_, err := svc.AcceptInvitation(context.Background(), uuid.New(), oid)
	if !errors.Is(err, apperr.ErrAlreadyMember) {
		t.Fatalf("expected ErrAlreadyMember, got %v", err)
	}
}

func TestAcceptInvitation_AlreadyMember(t *testing.T) {
	oid := ownerID()
	uid := otherID()
	cs := &mockClassStore{
		invitation: &domain.ClassInvitation{ClassID: uuid.New(), Role: domain.ClassMemberRoleStudent},
		getResult:  classItem(oid),
		isMember:   true,
	}
	svc := newSvc(cs, &mockCourseAccessor{})
	_, err := svc.AcceptInvitation(context.Background(), uuid.New(), uid)
	if !errors.Is(err, apperr.ErrAlreadyMember) {
		t.Fatalf("expected ErrAlreadyMember, got %v", err)
	}
}

func TestAcceptInvitation_Success(t *testing.T) {
	oid := ownerID()
	uid := otherID()
	classID := uuid.New()
	cs := &mockClassStore{
		invitation: &domain.ClassInvitation{ClassID: classID, Role: domain.ClassMemberRoleStudent},
		getResult:  classItem(oid),
	}
	svc := newSvc(cs, &mockCourseAccessor{})
	got, err := svc.AcceptInvitation(context.Background(), uuid.New(), uid)
	if err != nil || got != classID {
		t.Fatalf("unexpected: err=%v classID=%v", err, got)
	}
}

// --- GetInvitationDetail ---

func TestGetInvitationDetail_NotFound(t *testing.T) {
	svc := newSvc(&mockClassStore{}, &mockCourseAccessor{})
	_, err := svc.GetInvitationDetail(context.Background(), uuid.New())
	if !errors.Is(err, apperr.ErrInvitationNotFound) {
		t.Fatalf("expected ErrInvitationNotFound, got %v", err)
	}
}

func TestGetInvitationDetail_Success(t *testing.T) {
	detail := &domain.InvitationDetail{ClassTitle: "Математика", Role: domain.ClassMemberRoleStudent}
	cs := &mockClassStore{invDetail: detail}
	svc := newSvc(cs, &mockCourseAccessor{})
	got, err := svc.GetInvitationDetail(context.Background(), uuid.New())
	if err != nil || got.ClassTitle != "Математика" {
		t.Fatalf("unexpected: err=%v detail=%v", err, got)
	}
}

// --- RemoveMember ---

func TestRemoveMember_Forbidden(t *testing.T) {
	cs := &mockClassStore{getResult: classItem(ownerID())}
	svc := newSvc(cs, &mockCourseAccessor{})
	if err := svc.RemoveMember(context.Background(), uuid.New(), otherID(), uuid.New()); !errors.Is(err, apperr.ErrClassForbidden) {
		t.Fatalf("expected ErrClassForbidden, got %v", err)
	}
}

func TestRemoveMember_CannotRemoveOwner(t *testing.T) {
	oid := ownerID()
	cs := &mockClassStore{getResult: classItem(oid)}
	svc := newSvc(cs, &mockCourseAccessor{})
	if err := svc.RemoveMember(context.Background(), uuid.New(), oid, oid); !errors.Is(err, apperr.ErrClassRemoveOwner) {
		t.Fatalf("expected ErrClassRemoveOwner, got %v", err)
	}
}

func TestRemoveMember_Success(t *testing.T) {
	oid := ownerID()
	cs := &mockClassStore{getResult: classItem(oid)}
	svc := newSvc(cs, &mockCourseAccessor{})
	if err := svc.RemoveMember(context.Background(), uuid.New(), oid, otherID()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
