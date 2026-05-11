package attempt

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/pkg/apperr"
)

// --- mocks ---

type mockAttemptRepo struct {
	createID          uuid.UUID
	createErr         error
	byID              *domain.Attempt
	byIDErr           error
	upsertAnswerID    uuid.UUID
	upsertAnswerErr   error
	answers           []domain.AnswerSubmission
	answersErr        error
	evaluateErr       error
	publishErr        error
	setCompletedErr   error
	hasUngraded       bool
	hasUngradedErr    error
	bulkPublishErr    error
	setGradingErr     error
	setGradedErr      error
	insertEvalID      uuid.UUID
	insertEvalErr     error
	completeEvalErr   error
	failEvalErr       error
	evaluation        *domain.AnswerEvaluation
	evaluationErr     error
	hasPending        bool
	hasPendingErr     error
	updateScoreErr    error
	sumScores         float64
	sumScoresErr      error
	sessionAttempts   []domain.AttemptSummary
	sessionAttempsErr error
	answersWithQ      []domain.AnswerWithQuestion
	answersWithQErr   error
	submission        *domain.AnswerSubmission
	submissionErr     error
	userStatistic     *domain.UserStatistic
	userStatErr       error
	existingAttempt   *domain.Attempt
	existingErr       error
}

func (m *mockAttemptRepo) Create(_ context.Context, _, _ uuid.UUID, _ []uuid.UUID) (uuid.UUID, error) {
	return m.createID, m.createErr
}
func (m *mockAttemptRepo) GetByID(_ context.Context, _ uuid.UUID) (*domain.Attempt, error) {
	return m.byID, m.byIDErr
}
func (m *mockAttemptRepo) UpsertAnswer(_ context.Context, _, _ uuid.UUID, _ map[string]any) (uuid.UUID, error) {
	return m.upsertAnswerID, m.upsertAnswerErr
}
func (m *mockAttemptRepo) GetAnswers(_ context.Context, _ uuid.UUID) ([]domain.AnswerSubmission, error) {
	return m.answers, m.answersErr
}
func (m *mockAttemptRepo) EvaluateSubmission(_ context.Context, _ uuid.UUID, _ float64, _ domain.FinalSource, _ *string) error {
	return m.evaluateErr
}
func (m *mockAttemptRepo) Publish(_ context.Context, _ uuid.UUID, _, _ float64) error {
	return m.publishErr
}
func (m *mockAttemptRepo) SetCompleted(_ context.Context, _ uuid.UUID, _, _ float64) error {
	return m.setCompletedErr
}
func (m *mockAttemptRepo) HasUngradedFreeAnswers(_ context.Context, _ uuid.UUID) (bool, error) {
	return m.hasUngraded, m.hasUngradedErr
}
func (m *mockAttemptRepo) BulkPublishBySessionID(_ context.Context, _ uuid.UUID) error {
	return m.bulkPublishErr
}
func (m *mockAttemptRepo) FindExpiredInProgress(_ context.Context) ([]domain.Attempt, error) {
	return nil, nil
}
func (m *mockAttemptRepo) SetGrading(_ context.Context, _ uuid.UUID) error { return m.setGradingErr }
func (m *mockAttemptRepo) SetGraded(_ context.Context, _ uuid.UUID, _ float64) error {
	return m.setGradedErr
}
func (m *mockAttemptRepo) InsertEvaluation(_ context.Context, _ uuid.UUID, _ domain.EvaluationStatus, _ *float64, _ domain.FinalSource, _ *string) (uuid.UUID, error) {
	return m.insertEvalID, m.insertEvalErr
}
func (m *mockAttemptRepo) CompleteEvaluation(_ context.Context, _ uuid.UUID, _ float64, _ *string) error {
	return m.completeEvalErr
}
func (m *mockAttemptRepo) FailEvaluation(_ context.Context, _ uuid.UUID, _ string) error {
	return m.failEvalErr
}
func (m *mockAttemptRepo) GetEvaluationByID(_ context.Context, _ uuid.UUID) (*domain.AnswerEvaluation, error) {
	return m.evaluation, m.evaluationErr
}
func (m *mockAttemptRepo) HasPendingLLMEvaluations(_ context.Context, _ uuid.UUID) (bool, error) {
	return m.hasPending, m.hasPendingErr
}
func (m *mockAttemptRepo) UpdateSubmissionFinalScore(_ context.Context, _ uuid.UUID, _ float64, _ domain.FinalSource, _ *string) error {
	return m.updateScoreErr
}
func (m *mockAttemptRepo) SumScores(_ context.Context, _ uuid.UUID) (float64, error) {
	return m.sumScores, m.sumScoresErr
}
func (m *mockAttemptRepo) FindBySessionID(_ context.Context, _ uuid.UUID) ([]domain.AttemptSummary, error) {
	return m.sessionAttempts, m.sessionAttempsErr
}
func (m *mockAttemptRepo) GetAnswersWithQuestion(_ context.Context, _ uuid.UUID) ([]domain.AnswerWithQuestion, error) {
	return m.answersWithQ, m.answersWithQErr
}
func (m *mockAttemptRepo) GetSubmissionByID(_ context.Context, _ uuid.UUID) (*domain.AnswerSubmission, error) {
	return m.submission, m.submissionErr
}
func (m *mockAttemptRepo) GetUserStatistic(_ context.Context, _ uuid.UUID) (*domain.UserStatistic, error) {
	return m.userStatistic, m.userStatErr
}
func (m *mockAttemptRepo) GetBySessionAndUser(_ context.Context, _, _ uuid.UUID) (*domain.Attempt, error) {
	return m.existingAttempt, m.existingErr
}

type mockSessionReader struct {
	session          *domain.QuizSession
	sessionErr       error
	awaitingReview   []domain.AwaitingReviewSession
	awaitingErr      error
	recentGrades     []domain.RecentGradeItem
	recentGradesErr  error
	activeTests      []domain.ActiveTestItem
	activeTestsErr   error
}

func (m *mockSessionReader) GetByID(_ context.Context, _ uuid.UUID) (*domain.QuizSession, error) {
	return m.session, m.sessionErr
}
func (m *mockSessionReader) FindAwaitingReview(_ context.Context, _ uuid.UUID) ([]domain.AwaitingReviewSession, error) {
	return m.awaitingReview, m.awaitingErr
}
func (m *mockSessionReader) FindStudentRecentGrades(_ context.Context, _ uuid.UUID, _ int) ([]domain.RecentGradeItem, error) {
	return m.recentGrades, m.recentGradesErr
}
func (m *mockSessionReader) FindStudentActiveTests(_ context.Context, _ uuid.UUID) ([]domain.ActiveTestItem, error) {
	return m.activeTests, m.activeTestsErr
}

type mockQuizReader struct {
	quiz      *domain.QuizTemplate
	quizErr   error
	questions []domain.QuestionWithOptions
	questErr  error
}

func (m *mockQuizReader) GetByID(_ context.Context, _ uuid.UUID) (*domain.QuizTemplate, error) {
	return m.quiz, m.quizErr
}
func (m *mockQuizReader) GetQuestionsWithOptions(_ context.Context, _ uuid.UUID) ([]domain.QuestionWithOptions, error) {
	return m.questions, m.questErr
}

type mockSessionGrading struct{}

func (m *mockSessionGrading) FindFinishedNeedingGrading(_ context.Context) ([]domain.QuizSession, error) {
	return nil, nil
}
func (m *mockSessionGrading) SetGradingSent(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockSessionGrading) GetFreeAnswerSubmissionsForSession(_ context.Context, _ uuid.UUID) ([]domain.FreeAnswerSubmission, error) {
	return nil, nil
}
func (m *mockSessionGrading) FindSessionsReadyToAutoClose(_ context.Context) ([]domain.QuizSession, error) {
	return nil, nil
}
func (m *mockSessionGrading) UpdateStatus(_ context.Context, _ uuid.UUID, _ domain.SessionStatus) error {
	return nil
}

type mockQuestionReader struct {
	question *domain.Question
	err      error
}

func (m *mockQuestionReader) GetQuestionByID(_ context.Context, _ uuid.UUID) (*domain.Question, error) {
	return m.question, m.err
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

func newSvc(a *mockAttemptRepo, s *mockSessionReader, q *mockQuizReader) *Service {
	return NewService(a, s, &mockSessionGrading{}, q, &mockQuestionReader{}, &mockTx{}, &mockTaskSched{})
}

func activeTestSession() *domain.QuizSession {
	return &domain.QuizSession{
		ID:             uuid.New(),
		QuizTemplateID: uuid.New(),
		Mode:           domain.SessionModeTest,
		Status:         domain.SessionStatusActive,
	}
}

func libraryQuiz() *domain.QuizTemplate {
	return &domain.QuizTemplate{ID: uuid.New(), Source: domain.QuizSourceLibrary}
}

func inProgressAttempt(userID uuid.UUID) *domain.Attempt {
	return &domain.Attempt{
		ID:        uuid.New(),
		SessionID: uuid.New(),
		UserID:    userID,
		Status:    domain.AttemptStatusInProgress,
		StartedAt: time.Now(),
	}
}

// --- validateSessionForAttempt ---

func TestValidateSession_TestActive_OK(t *testing.T) {
	s := activeTestSession()
	if err := validateSessionForAttempt(s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSession_TestNotStarted(t *testing.T) {
	future := time.Now().Add(time.Hour)
	s := &domain.QuizSession{Mode: domain.SessionModeTest, StartedAt: &future}
	if err := validateSessionForAttempt(s); !errors.Is(err, apperr.ErrSessionNotStarted) {
		t.Fatalf("expected ErrSessionNotStarted, got %v", err)
	}
}

func TestValidateSession_TestDeadlinePassed(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	s := &domain.QuizSession{Mode: domain.SessionModeTest, FinishedAt: &past}
	if err := validateSessionForAttempt(s); !errors.Is(err, apperr.ErrSessionDeadlinePassed) {
		t.Fatalf("expected ErrSessionDeadlinePassed, got %v", err)
	}
}

func TestValidateSession_LiveWaiting_OK(t *testing.T) {
	s := &domain.QuizSession{Mode: domain.SessionModeLive, Status: domain.SessionStatusWaiting}
	if err := validateSessionForAttempt(s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSession_LiveNotWaiting(t *testing.T) {
	s := &domain.QuizSession{Mode: domain.SessionModeLive, Status: domain.SessionStatusRunning}
	if err := validateSessionForAttempt(s); !errors.Is(err, apperr.ErrSessionNotActive) {
		t.Fatalf("expected ErrSessionNotActive, got %v", err)
	}
}

// --- sanitizeQuestion ---

func TestSanitizeQuestion_SingleChoice_HidesIsCorrect(t *testing.T) {
	q := domain.QuestionWithOptions{
		Question: domain.Question{Type: domain.QuestionTypeSingleChoice},
		Options: []domain.AnswerOption{
			{ID: uuid.New(), Text: "A", IsCorrect: true},
			{ID: uuid.New(), Text: "B", IsCorrect: false},
		},
	}
	result := sanitizeQuestion(q)
	if len(result.Options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(result.Options))
	}
}

func TestSanitizeQuestion_Drag_ShufflesItems(t *testing.T) {
	q := domain.QuestionWithOptions{
		Question: domain.Question{
			Type:     domain.QuestionTypeDrag,
			Metadata: map[string]any{"correct_order": []any{"A", "B", "C"}},
		},
	}
	result := sanitizeQuestion(q)
	items, ok := result.Metadata["items"].([]any)
	if !ok || len(items) != 3 {
		t.Fatalf("expected shuffled items, got %v", result.Metadata)
	}
}

func TestSanitizeQuestion_Connection_ExposesLeftRight(t *testing.T) {
	q := domain.QuestionWithOptions{
		Question: domain.Question{
			Type: domain.QuestionTypeConnection,
			Metadata: map[string]any{
				"left":          []any{"A", "B"},
				"right":         []any{"1", "2"},
				"correct_pairs": map[string]any{"A": "1"},
			},
		},
	}
	result := sanitizeQuestion(q)
	if result.Metadata["correct_pairs"] != nil {
		t.Error("correct_pairs should not be exposed to student")
	}
	if result.Metadata["left"] == nil {
		t.Error("left should be exposed")
	}
}

func TestSanitizeQuestion_FreeAnswer_NoMeta(t *testing.T) {
	q := domain.QuestionWithOptions{
		Question: domain.Question{
			Type:     domain.QuestionTypeWithFreeAnswer,
			Metadata: map[string]any{"rubric": "detailed rubric"},
		},
	}
	result := sanitizeQuestion(q)
	if len(result.Metadata) != 0 {
		t.Errorf("expected no metadata for student, got %v", result.Metadata)
	}
}

// --- Create ---

func TestCreate_SessionNotFound(t *testing.T) {
	svc := newSvc(&mockAttemptRepo{}, &mockSessionReader{}, &mockQuizReader{})
	_, _, err := svc.Create(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, apperr.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestCreate_SessionNotStarted(t *testing.T) {
	future := time.Now().Add(time.Hour)
	sess := &domain.QuizSession{Mode: domain.SessionModeTest, StartedAt: &future}
	svc := newSvc(&mockAttemptRepo{}, &mockSessionReader{session: sess}, &mockQuizReader{})
	_, _, err := svc.Create(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, apperr.ErrSessionNotStarted) {
		t.Fatalf("expected ErrSessionNotStarted, got %v", err)
	}
}

func TestCreate_QuizNotFound(t *testing.T) {
	sess := activeTestSession()
	svc := newSvc(&mockAttemptRepo{}, &mockSessionReader{session: sess}, &mockQuizReader{})
	_, _, err := svc.Create(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, apperr.ErrQuizNotFound) {
		t.Fatalf("expected ErrQuizNotFound, got %v", err)
	}
}

func TestCreate_CourseTest_AlreadyExists(t *testing.T) {
	sess := activeTestSession()
	sess.Mode = domain.SessionModeTest
	quiz := &domain.QuizTemplate{Source: domain.QuizSourceCourse}
	existing := &domain.Attempt{ID: uuid.New()}
	svc := newSvc(
		&mockAttemptRepo{existingAttempt: existing},
		&mockSessionReader{session: sess},
		&mockQuizReader{quiz: quiz},
	)
	_, _, err := svc.Create(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, apperr.ErrAttemptAlreadyExists) {
		t.Fatalf("expected ErrAttemptAlreadyExists, got %v", err)
	}
}

func TestCreate_Success(t *testing.T) {
	sess := activeTestSession()
	quiz := libraryQuiz()
	id := uuid.New()
	a := &mockAttemptRepo{createID: id}
	svc := newSvc(a, &mockSessionReader{session: sess}, &mockQuizReader{quiz: quiz})
	attempt, questions, err := svc.Create(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempt.ID != id {
		t.Errorf("unexpected attempt ID: %v", attempt.ID)
	}
	_ = questions
}

// --- SubmitAnswer ---

func TestSubmitAnswer_NotFound(t *testing.T) {
	svc := newSvc(&mockAttemptRepo{}, &mockSessionReader{}, &mockQuizReader{})
	err := svc.SubmitAnswer(context.Background(), uuid.New(), uuid.New(), uuid.New(), nil)
	if !errors.Is(err, apperr.ErrAttemptNotFound) {
		t.Fatalf("expected ErrAttemptNotFound, got %v", err)
	}
}

func TestSubmitAnswer_WrongUser(t *testing.T) {
	attempt := inProgressAttempt(uuid.New())
	svc := newSvc(&mockAttemptRepo{byID: attempt}, &mockSessionReader{}, &mockQuizReader{})
	err := svc.SubmitAnswer(context.Background(), uuid.New(), uuid.New(), uuid.New(), nil)
	if !errors.Is(err, apperr.ErrAttemptForbidden) {
		t.Fatalf("expected ErrAttemptForbidden, got %v", err)
	}
}

func TestSubmitAnswer_NotActive(t *testing.T) {
	userID := uuid.New()
	attempt := inProgressAttempt(userID)
	attempt.Status = domain.AttemptStatusCompleted
	svc := newSvc(&mockAttemptRepo{byID: attempt}, &mockSessionReader{session: &domain.QuizSession{}}, &mockQuizReader{})
	err := svc.SubmitAnswer(context.Background(), uuid.New(), userID, uuid.New(), nil)
	if !errors.Is(err, apperr.ErrAttemptNotActive) {
		t.Fatalf("expected ErrAttemptNotActive, got %v", err)
	}
}

func TestSubmitAnswer_Success(t *testing.T) {
	userID := uuid.New()
	attempt := inProgressAttempt(userID)
	svc := newSvc(
		&mockAttemptRepo{byID: attempt},
		&mockSessionReader{session: &domain.QuizSession{}},
		&mockQuizReader{},
	)
	if err := svc.SubmitAnswer(context.Background(), uuid.New(), userID, uuid.New(), map[string]any{"text": "answer"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Finish ---

func TestFinish_NotFound(t *testing.T) {
	svc := newSvc(&mockAttemptRepo{}, &mockSessionReader{}, &mockQuizReader{})
	if err := svc.Finish(context.Background(), uuid.New(), uuid.New()); !errors.Is(err, apperr.ErrAttemptNotFound) {
		t.Fatalf("expected ErrAttemptNotFound, got %v", err)
	}
}

func TestFinish_WrongUser(t *testing.T) {
	attempt := inProgressAttempt(uuid.New())
	svc := newSvc(&mockAttemptRepo{byID: attempt}, &mockSessionReader{}, &mockQuizReader{})
	if err := svc.Finish(context.Background(), uuid.New(), uuid.New()); !errors.Is(err, apperr.ErrAttemptForbidden) {
		t.Fatalf("expected ErrAttemptForbidden, got %v", err)
	}
}

func TestFinish_NotActive(t *testing.T) {
	userID := uuid.New()
	attempt := inProgressAttempt(userID)
	attempt.Status = domain.AttemptStatusCompleted
	svc := newSvc(&mockAttemptRepo{byID: attempt}, &mockSessionReader{}, &mockQuizReader{})
	if err := svc.Finish(context.Background(), uuid.New(), userID); !errors.Is(err, apperr.ErrAttemptNotActive) {
		t.Fatalf("expected ErrAttemptNotActive, got %v", err)
	}
}

func TestFinish_Success_LibraryQuiz(t *testing.T) {
	userID := uuid.New()
	attempt := inProgressAttempt(userID)
	sess := &domain.QuizSession{QuizTemplateID: uuid.New(), MaxScore: 10}
	quiz := libraryQuiz()
	a := &mockAttemptRepo{byID: attempt}
	svc := newSvc(a, &mockSessionReader{session: sess}, &mockQuizReader{quiz: quiz})
	if err := svc.Finish(context.Background(), uuid.New(), userID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFinish_Success_NeedEvaluation(t *testing.T) {
	userID := uuid.New()
	attempt := inProgressAttempt(userID)
	sess := &domain.QuizSession{QuizTemplateID: uuid.New()}
	quiz := &domain.QuizTemplate{NeedEvaluation: true, Source: domain.QuizSourceLibrary}
	svc := newSvc(&mockAttemptRepo{byID: attempt}, &mockSessionReader{session: sess}, &mockQuizReader{quiz: quiz})
	if err := svc.Finish(context.Background(), uuid.New(), userID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- GetAttemptReview ---

func TestGetAttemptReview_NotFound(t *testing.T) {
	svc := newSvc(&mockAttemptRepo{}, &mockSessionReader{}, &mockQuizReader{})
	_, _, _, err := svc.GetAttemptReview(context.Background(), uuid.New(), nil)
	if !errors.Is(err, apperr.ErrAttemptNotFound) {
		t.Fatalf("expected ErrAttemptNotFound, got %v", err)
	}
}

func TestGetAttemptReview_NoAuth_UserAttempt_Unauthorized(t *testing.T) {
	attempt := &domain.Attempt{UserID: uuid.New(), Status: domain.AttemptStatusCompleted}
	svc := newSvc(&mockAttemptRepo{byID: attempt}, &mockSessionReader{}, &mockQuizReader{})
	_, _, _, err := svc.GetAttemptReview(context.Background(), uuid.New(), nil)
	if !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestGetAttemptReview_NoAuth_AnonAttempt_InProgress(t *testing.T) {
	attempt := &domain.Attempt{UserID: uuid.Nil, Status: domain.AttemptStatusInProgress}
	svc := newSvc(&mockAttemptRepo{byID: attempt}, &mockSessionReader{}, &mockQuizReader{})
	_, _, _, err := svc.GetAttemptReview(context.Background(), uuid.New(), nil)
	if !errors.Is(err, apperr.ErrAttemptNotActive) {
		t.Fatalf("expected ErrAttemptNotActive, got %v", err)
	}
}

func TestGetAttemptReview_Owner_InProgress(t *testing.T) {
	userID := uuid.New()
	attempt := &domain.Attempt{UserID: userID, Status: domain.AttemptStatusInProgress}
	svc := newSvc(&mockAttemptRepo{byID: attempt}, &mockSessionReader{}, &mockQuizReader{})
	_, _, _, err := svc.GetAttemptReview(context.Background(), uuid.New(), &userID)
	if !errors.Is(err, apperr.ErrAttemptNotActive) {
		t.Fatalf("expected ErrAttemptNotActive, got %v", err)
	}
}

func TestGetAttemptReview_Owner_Success(t *testing.T) {
	userID := uuid.New()
	attempt := &domain.Attempt{
		ID:        uuid.New(),
		SessionID: uuid.New(),
		UserID:    userID,
		Status:    domain.AttemptStatusCompleted,
		StartedAt: time.Now(),
	}
	sess := &domain.QuizSession{QuizTemplateID: uuid.New()}
	svc := newSvc(&mockAttemptRepo{byID: attempt}, &mockSessionReader{session: sess}, &mockQuizReader{})
	a, answers, enriched, err := svc.GetAttemptReview(context.Background(), uuid.New(), &userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a == nil || enriched {
		t.Errorf("expected attempt, enriched=false; got attempt=%v enriched=%v", a, enriched)
	}
	_ = answers
}

// --- GradeAttempt ---

func TestGradeAttempt_NegativeScore(t *testing.T) {
	svc := newSvc(&mockAttemptRepo{}, &mockSessionReader{}, &mockQuizReader{})
	grades := []domain.GradeItem{{SubmissionID: uuid.New(), Score: -1}}
	if err := svc.GradeAttempt(context.Background(), uuid.New(), uuid.New(), grades); !errors.Is(err, apperr.ErrScoreInvalid) {
		t.Fatalf("expected ErrScoreInvalid, got %v", err)
	}
}

// --- ListSessionAttempts ---

func TestListSessionAttempts_SessionNotFound(t *testing.T) {
	svc := newSvc(&mockAttemptRepo{}, &mockSessionReader{}, &mockQuizReader{})
	_, err := svc.ListSessionAttempts(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, apperr.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestListSessionAttempts_Success(t *testing.T) {
	sess := &domain.QuizSession{TeacherID: uuid.New(), QuizTemplateID: uuid.New()}
	quiz := &domain.QuizTemplate{AuthorID: sess.TeacherID}
	attempts := []domain.AttemptSummary{{ID: uuid.New(), Status: domain.AttemptStatusCompleted}}
	a := &mockAttemptRepo{sessionAttempts: attempts}
	svc := newSvc(a, &mockSessionReader{session: sess}, &mockQuizReader{quiz: quiz})
	result, err := svc.ListSessionAttempts(context.Background(), uuid.New(), sess.TeacherID)
	if err != nil || len(result) != 1 {
		t.Fatalf("unexpected: err=%v len=%d", err, len(result))
	}
}

// --- GetUserStatistic ---

func TestGetUserStatistic_Success(t *testing.T) {
	stat := &domain.UserStatistic{QuizCountPassed: 5}
	a := &mockAttemptRepo{userStatistic: stat}
	svc := newSvc(a, &mockSessionReader{}, &mockQuizReader{})
	result, err := svc.GetUserStatistic(context.Background(), uuid.New())
	if err != nil || result.QuizCountPassed != 5 {
		t.Fatalf("unexpected: err=%v stat=%v", err, result)
	}
}

// --- GetStudentDashboard ---

func TestGetStudentDashboard_Success(t *testing.T) {
	grades := []domain.RecentGradeItem{{SessionID: uuid.New()}}
	sess := &mockSessionReader{recentGrades: grades}
	svc := newSvc(&mockAttemptRepo{}, sess, &mockQuizReader{})
	dashboard, err := svc.GetStudentDashboard(context.Background(), uuid.New())
	if err != nil || len(dashboard.RecentGrades) != 1 {
		t.Fatalf("unexpected: err=%v", err)
	}
}

// --- ListAwaitingReview ---

func TestListAwaitingReview_Success(t *testing.T) {
	sessions := []domain.AwaitingReviewSession{{SessionID: uuid.New()}}
	sess := &mockSessionReader{awaitingReview: sessions}
	svc := newSvc(&mockAttemptRepo{}, sess, &mockQuizReader{})
	result, err := svc.ListAwaitingReview(context.Background(), uuid.New())
	if err != nil || len(result) != 1 {
		t.Fatalf("unexpected: err=%v len=%d", err, len(result))
	}
}

// --- PublishSession ---

func TestPublishSession_NotCompleted(t *testing.T) {
	sess := &domain.QuizSession{TeacherID: uuid.New(), QuizTemplateID: uuid.New()}
	quiz := &domain.QuizTemplate{AuthorID: sess.TeacherID}
	attempts := []domain.AttemptSummary{{Status: domain.AttemptStatusInProgress}}
	a := &mockAttemptRepo{sessionAttempts: attempts}
	svc := newSvc(a, &mockSessionReader{session: sess}, &mockQuizReader{quiz: quiz})
	if err := svc.PublishSession(context.Background(), uuid.New(), sess.TeacherID); !errors.Is(err, apperr.ErrAttemptNotCompleted) {
		t.Fatalf("expected ErrAttemptNotCompleted, got %v", err)
	}
}

func TestPublishSession_Success(t *testing.T) {
	tid := uuid.New()
	sess := &domain.QuizSession{TeacherID: tid, QuizTemplateID: uuid.New(), MaxScore: 10}
	quiz := &domain.QuizTemplate{AuthorID: tid}
	score := 8.0
	attempts := []domain.AttemptSummary{{
		ID: uuid.New(), Status: domain.AttemptStatusCompleted, UserID: uuid.New(), Score: &score,
	}}
	a := &mockAttemptRepo{sessionAttempts: attempts}
	svc := newSvc(a, &mockSessionReader{session: sess}, &mockQuizReader{quiz: quiz})
	if err := svc.PublishSession(context.Background(), uuid.New(), tid); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
