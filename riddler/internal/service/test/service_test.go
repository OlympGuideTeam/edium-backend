package test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/pkg/apperr"
)

type mockQuizRepo struct {
	quiz *domain.QuizTemplate
	err  error
}

func (m *mockQuizRepo) GetByID(_ context.Context, _ uuid.UUID) (*domain.QuizTemplate, error) {
	return m.quiz, m.err
}

type mockSessionRepo struct {
	createID  uuid.UUID
	createErr error
	session   *domain.QuizSession
	getErr    error
	updateErr error
}

func (m *mockSessionRepo) Create(_ context.Context, _ domain.CreateSessionParams) (uuid.UUID, error) {
	return m.createID, m.createErr
}
func (m *mockSessionRepo) GetByID(_ context.Context, _ uuid.UUID) (*domain.QuizSession, error) {
	return m.session, m.getErr
}
func (m *mockSessionRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ domain.SessionStatus) error {
	return m.updateErr
}

type mockTaskSched struct{ err error }

func (m *mockTaskSched) Schedule(_ context.Context, _ domain.TaskType, _ []byte) error {
	return m.err
}

type mockTx struct{ err error }

func (m *mockTx) WithTx(ctx context.Context, fn func(context.Context) error) error {
	if m.err != nil {
		return m.err
	}
	return fn(ctx)
}

type mockAttemptFinisher struct{ err error }

func (m *mockAttemptFinisher) FinishInProgressBySession(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func newSvc(q *mockQuizRepo, s *mockSessionRepo, task *mockTaskSched, tx *mockTx, a *mockAttemptFinisher) *Service {
	return NewService(q, s, task, tx, a)
}

func authorID() uuid.UUID { return uuid.MustParse("00000000-0000-0000-0000-000000000001") }
func otherID() uuid.UUID  { return uuid.MustParse("00000000-0000-0000-0000-000000000002") }

func courseQuiz(authorID uuid.UUID) *domain.QuizTemplate {
	courseID := uuid.New()
	return &domain.QuizTemplate{
		ID:       uuid.New(),
		AuthorID: authorID,
		Source:   domain.QuizSourceCourse,
		CourseID: &courseID,
	}
}

// --- CreateTestCourseSession ---

func TestCreateTestCourseSession_QuizNotFound(t *testing.T) {
	svc := newSvc(&mockQuizRepo{}, &mockSessionRepo{}, &mockTaskSched{}, &mockTx{}, &mockAttemptFinisher{})
	_, err := svc.CreateTestCourseSession(context.Background(), authorID(), uuid.New(), uuid.New(), domain.CreateTestCourseSessionParams{})
	if !errors.Is(err, apperr.ErrQuizNotFound) {
		t.Fatalf("expected ErrQuizNotFound, got %v", err)
	}
}

func TestCreateTestCourseSession_Forbidden_NotAuthor(t *testing.T) {
	quiz := courseQuiz(authorID())
	svc := newSvc(&mockQuizRepo{quiz: quiz}, &mockSessionRepo{}, &mockTaskSched{}, &mockTx{}, &mockAttemptFinisher{})
	_, err := svc.CreateTestCourseSession(context.Background(), otherID(), uuid.New(), uuid.New(), domain.CreateTestCourseSessionParams{})
	if !errors.Is(err, apperr.ErrQuizForbidden) {
		t.Fatalf("expected ErrQuizForbidden, got %v", err)
	}
}

func TestCreateTestCourseSession_Forbidden_NotCourseQuiz(t *testing.T) {
	quiz := &domain.QuizTemplate{AuthorID: authorID(), Source: domain.QuizSourceLibrary}
	svc := newSvc(&mockQuizRepo{quiz: quiz}, &mockSessionRepo{}, &mockTaskSched{}, &mockTx{}, &mockAttemptFinisher{})
	_, err := svc.CreateTestCourseSession(context.Background(), authorID(), uuid.New(), uuid.New(), domain.CreateTestCourseSessionParams{})
	if !errors.Is(err, apperr.ErrQuizForbidden) {
		t.Fatalf("expected ErrQuizForbidden, got %v", err)
	}
}

func TestCreateTestCourseSession_Success(t *testing.T) {
	aid := authorID()
	quiz := courseQuiz(aid)
	sessionID := uuid.New()
	sess := &mockSessionRepo{createID: sessionID}
	svc := newSvc(&mockQuizRepo{quiz: quiz}, sess, &mockTaskSched{}, &mockTx{}, &mockAttemptFinisher{})
	got, err := svc.CreateTestCourseSession(context.Background(), aid, uuid.New(), uuid.New(), domain.CreateTestCourseSessionParams{})
	if err != nil || got != sessionID {
		t.Fatalf("unexpected: err=%v id=%v", err, got)
	}
}

// --- FinishTestCourseSession ---

func TestFinishTestCourseSession_SessionNotFound(t *testing.T) {
	svc := newSvc(&mockQuizRepo{}, &mockSessionRepo{}, &mockTaskSched{}, &mockTx{}, &mockAttemptFinisher{})
	if err := svc.FinishTestCourseSession(context.Background(), authorID(), uuid.New()); !errors.Is(err, apperr.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestFinishTestCourseSession_Forbidden_NotTeacher(t *testing.T) {
	sess := &domain.QuizSession{
		TeacherID: authorID(),
		Source:    domain.LiveSourceCourse,
		Mode:      domain.SessionModeTest,
		Status:    domain.SessionStatusActive,
	}
	svc := newSvc(&mockQuizRepo{}, &mockSessionRepo{session: sess}, &mockTaskSched{}, &mockTx{}, &mockAttemptFinisher{})
	if err := svc.FinishTestCourseSession(context.Background(), otherID(), uuid.New()); !errors.Is(err, apperr.ErrQuizForbidden) {
		t.Fatalf("expected ErrQuizForbidden, got %v", err)
	}
}

func TestFinishTestCourseSession_Forbidden_WrongMode(t *testing.T) {
	aid := authorID()
	sess := &domain.QuizSession{
		TeacherID: aid,
		Source:    domain.LiveSourceCourse,
		Mode:      domain.SessionModeLive,
		Status:    domain.SessionStatusActive,
	}
	svc := newSvc(&mockQuizRepo{}, &mockSessionRepo{session: sess}, &mockTaskSched{}, &mockTx{}, &mockAttemptFinisher{})
	if err := svc.FinishTestCourseSession(context.Background(), aid, uuid.New()); !errors.Is(err, apperr.ErrQuizForbidden) {
		t.Fatalf("expected ErrQuizForbidden, got %v", err)
	}
}

func TestFinishTestCourseSession_AlreadyFinished(t *testing.T) {
	aid := authorID()
	sess := &domain.QuizSession{
		TeacherID: aid,
		Source:    domain.LiveSourceCourse,
		Mode:      domain.SessionModeTest,
		Status:    domain.SessionStatusFinished,
	}
	svc := newSvc(&mockQuizRepo{}, &mockSessionRepo{session: sess}, &mockTaskSched{}, &mockTx{}, &mockAttemptFinisher{})
	if err := svc.FinishTestCourseSession(context.Background(), aid, uuid.New()); !errors.Is(err, apperr.ErrSessionCompleted) {
		t.Fatalf("expected ErrSessionCompleted, got %v", err)
	}
}

func TestFinishTestCourseSession_FinishAttemptsError(t *testing.T) {
	aid := authorID()
	sess := &domain.QuizSession{
		TeacherID: aid,
		Source:    domain.LiveSourceCourse,
		Mode:      domain.SessionModeTest,
		Status:    domain.SessionStatusActive,
	}
	a := &mockAttemptFinisher{err: errors.New("db error")}
	svc := newSvc(&mockQuizRepo{}, &mockSessionRepo{session: sess}, &mockTaskSched{}, &mockTx{}, a)
	if err := svc.FinishTestCourseSession(context.Background(), aid, uuid.New()); err == nil {
		t.Fatal("expected error")
	}
}

func TestFinishTestCourseSession_Success(t *testing.T) {
	aid := authorID()
	sess := &domain.QuizSession{
		TeacherID: aid,
		Source:    domain.LiveSourceCourse,
		Mode:      domain.SessionModeTest,
		Status:    domain.SessionStatusActive,
	}
	svc := newSvc(&mockQuizRepo{}, &mockSessionRepo{session: sess}, &mockTaskSched{}, &mockTx{}, &mockAttemptFinisher{})
	if err := svc.FinishTestCourseSession(context.Background(), aid, uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
