package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"caesar/internal/domain"
)

// --- shared mocks ---

type mockTaskRepo struct {
	markDoneErr error
	scheduleErr error
}

func (m *mockTaskRepo) Schedule(_ context.Context, _ domain.TaskType, _ []byte) error {
	return m.scheduleErr
}
func (m *mockTaskRepo) ClaimPending(_ context.Context, _ domain.TaskType, _ int) ([]domain.Task, error) {
	return nil, nil
}
func (m *mockTaskRepo) MarkDone(_ context.Context, _ uuid.UUID) error { return m.markDoneErr }
func (m *mockTaskRepo) MarkFailed(_ context.Context, _ uuid.UUID, _ string, _ time.Duration) error {
	return nil
}

type mockUserStore struct {
	err error
}

func (m *mockUserStore) Create(_ context.Context, _ domain.User) error { return m.err }

type mockCourseItems struct {
	item           *domain.CourseItem
	findErr        error
	upsertErr      error
	updateScoreErr error
}

func (m *mockCourseItems) CreateItem(_ context.Context, _, _ uuid.UUID, _ domain.CourseItemType, _ json.RawMessage) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockCourseItems) FindItemByObjectID(_ context.Context, _ uuid.UUID) (*domain.CourseItem, error) {
	return m.item, m.findErr
}
func (m *mockCourseItems) DeleteItem(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockCourseItems) UpsertProgress(_ context.Context, _, _, _ uuid.UUID) error {
	return m.upsertErr
}
func (m *mockCourseItems) UpdateProgressScore(_ context.Context, _, _ uuid.UUID, _ float64) error {
	return m.updateScoreErr
}

type mockDraftStore struct {
	upsertID  uuid.UUID
	upsertErr error
	deleteErr error
}

func (m *mockDraftStore) UpsertCourseDraft(_ context.Context, _, _ uuid.UUID, _ string, _ json.RawMessage) (uuid.UUID, error) {
	return m.upsertID, m.upsertErr
}
func (m *mockDraftStore) DeleteDraftByTemplateAndCourse(_ context.Context, _, _ uuid.UUID) error {
	return m.deleteErr
}

type mockStudentStore struct {
	ids []uuid.UUID
	err error
}

func (m *mockStudentStore) GetStudentIDsByCourseID(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return m.ids, m.err
}

type mockCanceledStore struct {
	item      *domain.CourseItem
	findErr   error
	deleteErr error
	upsertErr error
}

func (m *mockCanceledStore) FindItemByObjectID(_ context.Context, _ uuid.UUID) (*domain.CourseItem, error) {
	return m.item, m.findErr
}
func (m *mockCanceledStore) DeleteItem(_ context.Context, _ uuid.UUID) error { return m.deleteErr }
func (m *mockCanceledStore) UpsertCourseDraft(_ context.Context, _, _ uuid.UUID, _ string, _ json.RawMessage) (uuid.UUID, error) {
	return uuid.New(), m.upsertErr
}

func makeTask(payload any) domain.Task {
	b, _ := json.Marshal(payload)
	return domain.Task{ID: uuid.New(), Payload: b}
}

// --- UserCreatedProcessor ---

func TestUserCreated_Success(t *testing.T) {
	w := NewUserCreatedProcessor(&mockTaskRepo{}, &mockUserStore{})
	t1 := makeTask(userCreatedPayload{
		UserID:  uuid.New().String(),
		Name:    "Ivan",
		Surname: "Petrov",
		Phone:   "+71234567890",
	})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUserCreated_InvalidUserID(t *testing.T) {
	w := NewUserCreatedProcessor(&mockTaskRepo{}, &mockUserStore{})
	t1 := makeTask(userCreatedPayload{UserID: "not-a-uuid"})
	if err := w.processTask(context.Background(), t1); err == nil {
		t.Fatal("expected error")
	}
}

func TestUserCreated_StoreError(t *testing.T) {
	w := NewUserCreatedProcessor(&mockTaskRepo{}, &mockUserStore{err: errors.New("db error")})
	t1 := makeTask(userCreatedPayload{UserID: uuid.New().String(), Name: "Ivan"})
	if err := w.processTask(context.Background(), t1); err == nil {
		t.Fatal("expected error")
	}
}

func TestUserCreated_BadPayload(t *testing.T) {
	w := NewUserCreatedProcessor(&mockTaskRepo{}, &mockUserStore{})
	if err := w.processTask(context.Background(), domain.Task{Payload: []byte("bad")}); err == nil {
		t.Fatal("expected error")
	}
}

// --- AttemptCreatedProcessor ---

func TestAttemptCreated_ItemNotFound_MarksDone(t *testing.T) {
	items := &mockCourseItems{item: nil}
	w := NewAttemptCreatedProcessor(&mockTaskRepo{}, items)
	t1 := makeTask(attemptCreatedPayload{
		AttemptID: uuid.New().String(),
		SessionID: uuid.New().String(),
		UserID:    uuid.New().String(),
	})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAttemptCreated_Success(t *testing.T) {
	item := &domain.CourseItem{ID: uuid.New()}
	items := &mockCourseItems{item: item}
	w := NewAttemptCreatedProcessor(&mockTaskRepo{}, items)
	t1 := makeTask(attemptCreatedPayload{
		AttemptID: uuid.New().String(),
		SessionID: uuid.New().String(),
		UserID:    uuid.New().String(),
	})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAttemptCreated_UpsertProgressError(t *testing.T) {
	item := &domain.CourseItem{ID: uuid.New()}
	items := &mockCourseItems{item: item, upsertErr: errors.New("db error")}
	w := NewAttemptCreatedProcessor(&mockTaskRepo{}, items)
	t1 := makeTask(attemptCreatedPayload{
		AttemptID: uuid.New().String(),
		SessionID: uuid.New().String(),
		UserID:    uuid.New().String(),
	})
	if err := w.processTask(context.Background(), t1); err == nil {
		t.Fatal("expected error")
	}
}

func TestAttemptCreated_BadPayload(t *testing.T) {
	w := NewAttemptCreatedProcessor(&mockTaskRepo{}, &mockCourseItems{})
	if err := w.processTask(context.Background(), domain.Task{Payload: []byte("bad")}); err == nil {
		t.Fatal("expected error")
	}
}

// --- AttemptScoredProcessor ---

func TestAttemptScored_ItemNotFound_MarksDone(t *testing.T) {
	items := &mockCourseItems{item: nil}
	w := NewAttemptScoredProcessor(&mockTaskRepo{}, items)
	t1 := makeTask(attemptScoredPayload{
		SessionID:  uuid.New().String(),
		UserID:     uuid.New().String(),
		TotalScore: 8,
		MaxScore:   10,
	})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAttemptScored_Success(t *testing.T) {
	item := &domain.CourseItem{ID: uuid.New()}
	items := &mockCourseItems{item: item}
	w := NewAttemptScoredProcessor(&mockTaskRepo{}, items)
	t1 := makeTask(attemptScoredPayload{
		SessionID:  uuid.New().String(),
		UserID:     uuid.New().String(),
		TotalScore: 7.5,
		MaxScore:   10,
	})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAttemptScored_ZeroMaxScore(t *testing.T) {
	item := &domain.CourseItem{ID: uuid.New()}
	items := &mockCourseItems{item: item}
	w := NewAttemptScoredProcessor(&mockTaskRepo{}, items)
	t1 := makeTask(attemptScoredPayload{
		SessionID: uuid.New().String(),
		UserID:    uuid.New().String(),
		MaxScore:  0,
	})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- QuizTemplateAttachedProcessor ---

func TestQuizTemplateAttached_Success(t *testing.T) {
	drafts := &mockDraftStore{upsertID: uuid.New()}
	w := NewQuizTemplateAttachedProcessor(&mockTaskRepo{}, drafts)
	t1 := makeTask(quizTemplateAttachedPayload{
		QuizTemplateID: uuid.New().String(),
		CourseID:       uuid.New().String(),
		Title:          "Quiz 1",
	})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestQuizTemplateAttached_UpsertError(t *testing.T) {
	drafts := &mockDraftStore{upsertErr: errors.New("db error")}
	w := NewQuizTemplateAttachedProcessor(&mockTaskRepo{}, drafts)
	t1 := makeTask(quizTemplateAttachedPayload{
		QuizTemplateID: uuid.New().String(),
		CourseID:       uuid.New().String(),
		Title:          "Quiz 1",
	})
	if err := w.processTask(context.Background(), t1); err == nil {
		t.Fatal("expected error")
	}
}

func TestQuizTemplateAttached_BadPayload(t *testing.T) {
	w := NewQuizTemplateAttachedProcessor(&mockTaskRepo{}, &mockDraftStore{})
	if err := w.processTask(context.Background(), domain.Task{Payload: []byte("bad")}); err == nil {
		t.Fatal("expected error")
	}
}

// --- CourseSessionCreatedProcessor ---

func TestCourseSessionCreated_NoTemplate(t *testing.T) {
	items := &mockCourseItems{}
	w := NewCourseSessionCreatedProcessor(&mockTaskRepo{}, items, &mockDraftStore{}, &mockStudentStore{})
	t1 := makeTask(courseSessionCreatedPayload{
		SessionID: uuid.New().String(),
		ModuleID:  uuid.New().String(),
		Title:     "Quiz",
		Mode:      "live",
	})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCourseSessionCreated_WithTemplate_StudentsNotified(t *testing.T) {
	templateID := uuid.New().String()
	courseIDStr := uuid.New().String()
	items := &mockCourseItems{}
	students := &mockStudentStore{ids: []uuid.UUID{uuid.New(), uuid.New()}}
	w := NewCourseSessionCreatedProcessor(&mockTaskRepo{}, items, &mockDraftStore{}, students)
	t1 := makeTask(courseSessionCreatedPayload{
		SessionID:      uuid.New().String(),
		ModuleID:       uuid.New().String(),
		QuizTemplateID: &templateID,
		CourseID:       &courseIDStr,
		Title:          "Quiz",
		Mode:           "live",
	})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCourseSessionCreated_BadPayload(t *testing.T) {
	w := NewCourseSessionCreatedProcessor(&mockTaskRepo{}, &mockCourseItems{}, &mockDraftStore{}, &mockStudentStore{})
	if err := w.processTask(context.Background(), domain.Task{Payload: []byte("bad")}); err == nil {
		t.Fatal("expected error")
	}
}

// --- CourseSessionCanceledProcessor ---

func TestCourseSessionCanceled_ItemNotFound(t *testing.T) {
	store := &mockCanceledStore{item: nil}
	w := NewCourseSessionCanceledProcessor(&mockTaskRepo{}, store)
	courseIDStr := uuid.New().String()
	t1 := makeTask(courseSessionCanceledPayload{
		SessionID:      uuid.New().String(),
		QuizTemplateID: uuid.New().String(),
		CourseID:       &courseIDStr,
		Title:          "Quiz",
	})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCourseSessionCanceled_ItemFound_DeletesAndRestoresDraft(t *testing.T) {
	item := &domain.CourseItem{ID: uuid.New()}
	store := &mockCanceledStore{item: item}
	w := NewCourseSessionCanceledProcessor(&mockTaskRepo{}, store)
	courseIDStr := uuid.New().String()
	t1 := makeTask(courseSessionCanceledPayload{
		SessionID:      uuid.New().String(),
		QuizTemplateID: uuid.New().String(),
		CourseID:       &courseIDStr,
		Title:          "Quiz",
	})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCourseSessionCanceled_NoCourseID(t *testing.T) {
	store := &mockCanceledStore{}
	w := NewCourseSessionCanceledProcessor(&mockTaskRepo{}, store)
	t1 := makeTask(courseSessionCanceledPayload{
		SessionID:      uuid.New().String(),
		QuizTemplateID: uuid.New().String(),
		Title:          "Quiz",
	})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCourseSessionCanceled_BadPayload(t *testing.T) {
	w := NewCourseSessionCanceledProcessor(&mockTaskRepo{}, &mockCanceledStore{})
	if err := w.processTask(context.Background(), domain.Task{Payload: []byte("bad")}); err == nil {
		t.Fatal("expected error")
	}
}
