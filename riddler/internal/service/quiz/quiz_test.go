package quiz

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/pkg/apperr"
)

// --- mocks ---

type mockQuizRepo struct {
	quiz           *domain.QuizTemplate
	getErr         error
	createID       uuid.UUID
	createErr      error
	updateErr      error
	deleteErr      error
	publishErr     error
	setLibErr      error
	setNeedEvalErr error
	hasFreeAnswer  bool
	hasFreeAnsErr  error
	questions      []domain.QuestionWithOptions
	questionsErr   error
	addQuestionID  uuid.UUID
	addQuestionIdx int
	addQuestionErr error
	deleteQErr     error
	reorderErr     error
	listPublished  []domain.QuizListItem
	listPubErr     error
	listByAuthor   []domain.QuizListItem
	listByAuthErr  error
	copyID         uuid.UUID
	copyErr        error
}

func (m *mockQuizRepo) Create(_ context.Context, _ uuid.UUID, _ string, _ *string, _ domain.QuizDefaultSettings, _ domain.QuizSource, _ *uuid.UUID) (uuid.UUID, error) {
	return m.createID, m.createErr
}
func (m *mockQuizRepo) GetByID(_ context.Context, _ uuid.UUID) (*domain.QuizTemplate, error) {
	return m.quiz, m.getErr
}
func (m *mockQuizRepo) AddQuestion(_ context.Context, _ domain.AddQuestionParams) (uuid.UUID, int, error) {
	return m.addQuestionID, m.addQuestionIdx, m.addQuestionErr
}
func (m *mockQuizRepo) Update(_ context.Context, _ uuid.UUID, _, _ *string, _ *domain.QuizDefaultSettings) error {
	return m.updateErr
}
func (m *mockQuizRepo) ReorderQuestions(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return m.reorderErr
}
func (m *mockQuizRepo) DeleteQuestion(_ context.Context, _, _ uuid.UUID) error { return m.deleteQErr }
func (m *mockQuizRepo) Delete(_ context.Context, _ uuid.UUID) error            { return m.deleteErr }
func (m *mockQuizRepo) Publish(_ context.Context, _ uuid.UUID) error           { return m.publishErr }
func (m *mockQuizRepo) SetLibrarySession(_ context.Context, _, _ uuid.UUID) error {
	return m.setLibErr
}
func (m *mockQuizRepo) GetQuestionsWithOptions(_ context.Context, _ uuid.UUID) ([]domain.QuestionWithOptions, error) {
	return m.questions, m.questionsErr
}
func (m *mockQuizRepo) SetNeedEvaluation(_ context.Context, _ uuid.UUID, _ bool) error {
	return m.setNeedEvalErr
}
func (m *mockQuizRepo) HasFreeAnswerQuestions(_ context.Context, _ uuid.UUID) (bool, error) {
	return m.hasFreeAnswer, m.hasFreeAnsErr
}
func (m *mockQuizRepo) ListPublished(_ context.Context, _ bool, _ *string) ([]domain.QuizListItem, error) {
	return m.listPublished, m.listPubErr
}
func (m *mockQuizRepo) ListByAuthor(_ context.Context, _ uuid.UUID, _ *string) ([]domain.QuizListItem, error) {
	return m.listByAuthor, m.listByAuthErr
}
func (m *mockQuizRepo) Copy(_ context.Context, _, _ uuid.UUID, _ domain.QuizSource, _ *uuid.UUID) (uuid.UUID, error) {
	return m.copyID, m.copyErr
}

type mockSessionSvc struct {
	createID    uuid.UUID
	createErr   error
	session     *domain.QuizSession
	getErr      error
	hasAttempts bool
	hasAttErr   error
	hasSessions bool
	hasSessErr  error
	deleteErr   error
}

func (m *mockSessionSvc) Create(_ context.Context, _ domain.CreateSessionParams) (uuid.UUID, error) {
	return m.createID, m.createErr
}
func (m *mockSessionSvc) GetByID(_ context.Context, _ uuid.UUID) (*domain.QuizSession, error) {
	return m.session, m.getErr
}
func (m *mockSessionSvc) HasAttempts(_ context.Context, _ uuid.UUID) (bool, error) {
	return m.hasAttempts, m.hasAttErr
}
func (m *mockSessionSvc) HasSessions(_ context.Context, _ uuid.UUID) (bool, error) {
	return m.hasSessions, m.hasSessErr
}
func (m *mockSessionSvc) Delete(_ context.Context, _ uuid.UUID) error { return m.deleteErr }

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

type mockAttemptAccessor struct {
	attempts    []domain.QuizAttemptSummary
	attErr      error
	batchResult map[uuid.UUID][]domain.QuizAttemptSummary
	batchErr    error
}

func (m *mockAttemptAccessor) GetQuizAttemptsByUser(_ context.Context, _, _ uuid.UUID) ([]domain.QuizAttemptSummary, error) {
	return m.attempts, m.attErr
}
func (m *mockAttemptAccessor) GetAttemptsByUserBatch(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (map[uuid.UUID][]domain.QuizAttemptSummary, error) {
	return m.batchResult, m.batchErr
}

func authorID() uuid.UUID { return uuid.MustParse("00000000-0000-0000-0000-000000000001") }
func otherID() uuid.UUID  { return uuid.MustParse("00000000-0000-0000-0000-000000000002") }

func activeQuiz(authorID uuid.UUID) *domain.QuizTemplate {
	return &domain.QuizTemplate{
		ID:       uuid.New(),
		AuthorID: authorID,
		Title:    "Тест по математике",
		Source:   domain.QuizSourceLibrary,
	}
}

func newSvc(q *mockQuizRepo, a *mockAttemptAccessor, s *mockSessionSvc, tx *mockTx, t *mockTaskSched) *Service {
	return NewService(q, a, s, tx, t)
}

// --- CreateQuiz ---

func TestCreateQuiz_EmptyTitle(t *testing.T) {
	svc := newSvc(&mockQuizRepo{}, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	_, err := svc.CreateQuiz(context.Background(), authorID(), "   ", nil, domain.QuizDefaultSettings{}, nil)
	if !errors.Is(err, apperr.ErrQuizEmptyTitle) {
		t.Fatalf("expected ErrQuizEmptyTitle, got %v", err)
	}
}

func TestCreateQuiz_StoreError(t *testing.T) {
	q := &mockQuizRepo{createErr: errors.New("db error")}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	_, err := svc.CreateQuiz(context.Background(), authorID(), "Title", nil, domain.QuizDefaultSettings{}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateQuiz_Success_Library(t *testing.T) {
	id := uuid.New()
	q := &mockQuizRepo{createID: id}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	got, err := svc.CreateQuiz(context.Background(), authorID(), "Title", nil, domain.QuizDefaultSettings{}, nil)
	if err != nil || got != id {
		t.Fatalf("unexpected: err=%v id=%v", err, got)
	}
}

func TestCreateQuiz_Success_WithCourse(t *testing.T) {
	id := uuid.New()
	courseID := uuid.New()
	q := &mockQuizRepo{createID: id}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	got, err := svc.CreateQuiz(context.Background(), authorID(), "Title", nil, domain.QuizDefaultSettings{}, &courseID)
	if err != nil || got != id {
		t.Fatalf("unexpected: err=%v id=%v", err, got)
	}
}

// --- GetQuiz ---

func TestGetQuiz_NotFound(t *testing.T) {
	svc := newSvc(&mockQuizRepo{}, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	_, err := svc.GetQuiz(context.Background(), uuid.New(), authorID())
	if !errors.Is(err, apperr.ErrQuizNotFound) {
		t.Fatalf("expected ErrQuizNotFound, got %v", err)
	}
}

func TestGetQuiz_Private_OtherUser(t *testing.T) {
	q := &mockQuizRepo{quiz: &domain.QuizTemplate{AuthorID: authorID(), IsPublic: false}}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	_, err := svc.GetQuiz(context.Background(), uuid.New(), otherID())
	if !errors.Is(err, apperr.ErrQuizForbidden) {
		t.Fatalf("expected ErrQuizForbidden, got %v", err)
	}
}

func TestGetQuiz_Public_AnyUser(t *testing.T) {
	q := &mockQuizRepo{quiz: &domain.QuizTemplate{AuthorID: authorID(), IsPublic: true}}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	detail, err := svc.GetQuiz(context.Background(), uuid.New(), otherID())
	if err != nil || detail == nil {
		t.Fatalf("unexpected: err=%v", err)
	}
}

func TestGetQuiz_Author(t *testing.T) {
	aid := authorID()
	q := &mockQuizRepo{quiz: &domain.QuizTemplate{AuthorID: aid, IsPublic: false}}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	detail, err := svc.GetQuiz(context.Background(), uuid.New(), aid)
	if err != nil || detail == nil {
		t.Fatalf("unexpected: err=%v", err)
	}
}

// --- GetQuizForStudent ---

func TestGetQuizForStudent_NotFound(t *testing.T) {
	svc := newSvc(&mockQuizRepo{}, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	_, err := svc.GetQuizForStudent(context.Background(), uuid.New(), authorID())
	if !errors.Is(err, apperr.ErrQuizNotFound) {
		t.Fatalf("expected ErrQuizNotFound, got %v", err)
	}
}

func TestGetQuizForStudent_NotPublic(t *testing.T) {
	q := &mockQuizRepo{quiz: &domain.QuizTemplate{IsPublic: false}}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	_, err := svc.GetQuizForStudent(context.Background(), uuid.New(), authorID())
	if !errors.Is(err, apperr.ErrQuizNotAvailable) {
		t.Fatalf("expected ErrQuizNotAvailable, got %v", err)
	}
}

func TestGetQuizForStudent_Success(t *testing.T) {
	q := &mockQuizRepo{quiz: &domain.QuizTemplate{IsPublic: true}}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	view, err := svc.GetQuizForStudent(context.Background(), uuid.New(), authorID())
	if err != nil || view == nil {
		t.Fatalf("unexpected: err=%v", err)
	}
}

// --- UpdateQuiz ---

func TestUpdateQuiz_EmptyTitle(t *testing.T) {
	empty := "   "
	svc := newSvc(&mockQuizRepo{}, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	err := svc.UpdateQuiz(context.Background(), uuid.New(), authorID(), &empty, nil, nil)
	if !errors.Is(err, apperr.ErrQuizEmptyTitle) {
		t.Fatalf("expected ErrQuizEmptyTitle, got %v", err)
	}
}

func TestUpdateQuiz_NotFound(t *testing.T) {
	svc := newSvc(&mockQuizRepo{}, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	title := "New"
	err := svc.UpdateQuiz(context.Background(), uuid.New(), authorID(), &title, nil, nil)
	if !errors.Is(err, apperr.ErrQuizNotFound) {
		t.Fatalf("expected ErrQuizNotFound, got %v", err)
	}
}

func TestUpdateQuiz_Forbidden(t *testing.T) {
	q := &mockQuizRepo{quiz: activeQuiz(authorID())}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	title := "New"
	err := svc.UpdateQuiz(context.Background(), uuid.New(), otherID(), &title, nil, nil)
	if !errors.Is(err, apperr.ErrQuizForbidden) {
		t.Fatalf("expected ErrQuizForbidden, got %v", err)
	}
}

func TestUpdateQuiz_Success(t *testing.T) {
	aid := authorID()
	q := &mockQuizRepo{quiz: activeQuiz(aid)}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	title := "New Title"
	if err := svc.UpdateQuiz(context.Background(), uuid.New(), aid, &title, nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- PublishQuiz ---

func TestPublishQuiz_NotFound(t *testing.T) {
	svc := newSvc(&mockQuizRepo{}, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	if err := svc.PublishQuiz(context.Background(), uuid.New(), authorID()); !errors.Is(err, apperr.ErrQuizNotFound) {
		t.Fatalf("expected ErrQuizNotFound, got %v", err)
	}
}

func TestPublishQuiz_Forbidden(t *testing.T) {
	q := &mockQuizRepo{quiz: activeQuiz(authorID())}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	if err := svc.PublishQuiz(context.Background(), uuid.New(), otherID()); !errors.Is(err, apperr.ErrQuizForbidden) {
		t.Fatalf("expected ErrQuizForbidden, got %v", err)
	}
}

func TestPublishQuiz_CourseOnly(t *testing.T) {
	aid := authorID()
	quiz := activeQuiz(aid)
	quiz.Source = domain.QuizSourceCourse
	q := &mockQuizRepo{quiz: quiz}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	if err := svc.PublishQuiz(context.Background(), uuid.New(), aid); !errors.Is(err, apperr.ErrQuizCourseOnly) {
		t.Fatalf("expected ErrQuizCourseOnly, got %v", err)
	}
}

func TestPublishQuiz_AlreadyPublished(t *testing.T) {
	aid := authorID()
	quiz := activeQuiz(aid)
	quiz.IsPublic = true
	q := &mockQuizRepo{quiz: quiz}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	if err := svc.PublishQuiz(context.Background(), uuid.New(), aid); !errors.Is(err, apperr.ErrQuizAlreadyPublished) {
		t.Fatalf("expected ErrQuizAlreadyPublished, got %v", err)
	}
}

func TestPublishQuiz_Success(t *testing.T) {
	aid := authorID()
	quiz := activeQuiz(aid)
	sessionID := uuid.New()
	q := &mockQuizRepo{quiz: quiz}
	s := &mockSessionSvc{createID: sessionID}
	svc := newSvc(q, &mockAttemptAccessor{}, s, &mockTx{}, &mockTaskSched{})
	if err := svc.PublishQuiz(context.Background(), uuid.New(), aid); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPublishQuiz_NeedEvaluation_SkipsSession(t *testing.T) {
	aid := authorID()
	quiz := activeQuiz(aid)
	quiz.NeedEvaluation = true
	q := &mockQuizRepo{quiz: quiz}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	if err := svc.PublishQuiz(context.Background(), uuid.New(), aid); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- DeleteQuiz ---

func TestDeleteQuiz_NotFound(t *testing.T) {
	svc := newSvc(&mockQuizRepo{}, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	if err := svc.DeleteQuiz(context.Background(), uuid.New(), authorID()); !errors.Is(err, apperr.ErrQuizNotFound) {
		t.Fatalf("expected ErrQuizNotFound, got %v", err)
	}
}

func TestDeleteQuiz_Forbidden(t *testing.T) {
	q := &mockQuizRepo{quiz: activeQuiz(authorID())}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	if err := svc.DeleteQuiz(context.Background(), uuid.New(), otherID()); !errors.Is(err, apperr.ErrQuizForbidden) {
		t.Fatalf("expected ErrQuizForbidden, got %v", err)
	}
}

func TestDeleteQuiz_IsPublic(t *testing.T) {
	aid := authorID()
	quiz := activeQuiz(aid)
	quiz.IsPublic = true
	q := &mockQuizRepo{quiz: quiz}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	if err := svc.DeleteQuiz(context.Background(), uuid.New(), aid); !errors.Is(err, apperr.ErrQuizIsPublic) {
		t.Fatalf("expected ErrQuizIsPublic, got %v", err)
	}
}

func TestDeleteQuiz_HasSessions(t *testing.T) {
	aid := authorID()
	q := &mockQuizRepo{quiz: activeQuiz(aid)}
	s := &mockSessionSvc{hasSessions: true}
	svc := newSvc(q, &mockAttemptAccessor{}, s, &mockTx{}, &mockTaskSched{})
	if err := svc.DeleteQuiz(context.Background(), uuid.New(), aid); !errors.Is(err, apperr.ErrQuizHasSessions) {
		t.Fatalf("expected ErrQuizHasSessions, got %v", err)
	}
}

func TestDeleteQuiz_Success(t *testing.T) {
	aid := authorID()
	q := &mockQuizRepo{quiz: activeQuiz(aid)}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	if err := svc.DeleteQuiz(context.Background(), uuid.New(), aid); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- ListQuizzes ---

func TestListQuizzes_Teacher(t *testing.T) {
	items := []domain.QuizListItem{{ID: uuid.New(), Title: "Math"}}
	q := &mockQuizRepo{listPublished: items}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	result, err := svc.ListQuizzes(context.Background(), domain.RoleTeacher, nil, nil)
	if err != nil || len(result) != 1 {
		t.Fatalf("unexpected: err=%v len=%d", err, len(result))
	}
}

func TestListQuizzes_Student_WithAttempts(t *testing.T) {
	quizID := uuid.New()
	items := []domain.QuizListItem{{ID: quizID}}
	userID := authorID()
	q := &mockQuizRepo{listPublished: items}
	a := &mockAttemptAccessor{batchResult: map[uuid.UUID][]domain.QuizAttemptSummary{
		quizID: {{ID: uuid.New(), Status: domain.AttemptStatusPublished}},
	}}
	svc := newSvc(q, a, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	result, err := svc.ListQuizzes(context.Background(), domain.RoleStudent, &userID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result[0].Attempts) != 1 {
		t.Errorf("expected 1 attempt, got %d", len(result[0].Attempts))
	}
}

// --- ListMyQuizzes ---

func TestListMyQuizzes_Success(t *testing.T) {
	items := []domain.QuizListItem{{ID: uuid.New()}}
	q := &mockQuizRepo{listByAuthor: items}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	result, err := svc.ListMyQuizzes(context.Background(), authorID(), nil)
	if err != nil || len(result) != 1 {
		t.Fatalf("unexpected: err=%v len=%d", err, len(result))
	}
}

// --- DeleteQuestion ---

func TestDeleteQuestion_NotFound(t *testing.T) {
	svc := newSvc(&mockQuizRepo{}, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	if err := svc.DeleteQuestion(context.Background(), uuid.New(), uuid.New(), authorID()); !errors.Is(err, apperr.ErrQuizNotFound) {
		t.Fatalf("expected ErrQuizNotFound, got %v", err)
	}
}

func TestDeleteQuestion_Forbidden(t *testing.T) {
	q := &mockQuizRepo{quiz: activeQuiz(authorID())}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	if err := svc.DeleteQuestion(context.Background(), uuid.New(), uuid.New(), otherID()); !errors.Is(err, apperr.ErrQuizForbidden) {
		t.Fatalf("expected ErrQuizForbidden, got %v", err)
	}
}

func TestDeleteQuestion_Success(t *testing.T) {
	aid := authorID()
	q := &mockQuizRepo{quiz: activeQuiz(aid)}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	if err := svc.DeleteQuestion(context.Background(), uuid.New(), uuid.New(), aid); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteQuestion_NeedEval_NoFreeAnswers_ClearsFlag(t *testing.T) {
	aid := authorID()
	quiz := activeQuiz(aid)
	quiz.NeedEvaluation = true
	q := &mockQuizRepo{quiz: quiz, hasFreeAnswer: false}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	if err := svc.DeleteQuestion(context.Background(), uuid.New(), uuid.New(), aid); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- ReorderQuestions ---

func TestReorderQuestions_Forbidden(t *testing.T) {
	q := &mockQuizRepo{quiz: activeQuiz(authorID())}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	if err := svc.ReorderQuestions(context.Background(), uuid.New(), otherID(), nil); !errors.Is(err, apperr.ErrQuizForbidden) {
		t.Fatalf("expected ErrQuizForbidden, got %v", err)
	}
}

func TestReorderQuestions_Success(t *testing.T) {
	aid := authorID()
	q := &mockQuizRepo{quiz: activeQuiz(aid)}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	if err := svc.ReorderQuestions(context.Background(), uuid.New(), aid, []uuid.UUID{uuid.New()}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- GenerateQuestions ---

func TestGenerateQuestions_Forbidden(t *testing.T) {
	q := &mockQuizRepo{quiz: activeQuiz(authorID())}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	_, err := svc.GenerateQuestions(context.Background(), uuid.New(), otherID(), "text")
	if !errors.Is(err, apperr.ErrQuizForbidden) {
		t.Fatalf("expected ErrQuizForbidden, got %v", err)
	}
}

func TestGenerateQuestions_Success(t *testing.T) {
	aid := authorID()
	q := &mockQuizRepo{quiz: activeQuiz(aid)}
	svc := newSvc(q, &mockAttemptAccessor{}, &mockSessionSvc{}, &mockTx{}, &mockTaskSched{})
	jobID, err := svc.GenerateQuestions(context.Background(), uuid.New(), aid, "some text")
	if err != nil || jobID == uuid.Nil {
		t.Fatalf("unexpected: err=%v id=%v", err, jobID)
	}
}
