package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

// --- shared mocks ---

type mockTaskRepo struct {
	markDoneErr error
	scheduleErr error
}

func (m *mockTaskRepo) ClaimPending(_ context.Context, _ domain.TaskType, _ int) ([]domain.Task, error) {
	return nil, nil
}
func (m *mockTaskRepo) MarkDone(_ context.Context, _ uuid.UUID) error { return m.markDoneErr }
func (m *mockTaskRepo) MarkFailed(_ context.Context, _ uuid.UUID, _ string, _ time.Duration) error {
	return nil
}
func (m *mockTaskRepo) Schedule(_ context.Context, _ domain.TaskType, _ []byte) error {
	return m.scheduleErr
}

func makeTask(payload any) domain.Task {
	b, _ := json.Marshal(payload)
	return domain.Task{ID: uuid.New(), Payload: b}
}

// --- CourseSessionDeletedProcessor ---

type mockSessionDeleter struct{ err error }

func (m *mockSessionDeleter) Delete(_ context.Context, _ uuid.UUID) error { return m.err }

func TestCourseSessionDeleted_Success(t *testing.T) {
	w := NewCourseSessionDeletedProcessor(&mockTaskRepo{}, &mockSessionDeleter{})
	t1 := makeTask(courseSessionDeletedPayload{SessionID: uuid.New()})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCourseSessionDeleted_DeleteError(t *testing.T) {
	w := NewCourseSessionDeletedProcessor(&mockTaskRepo{}, &mockSessionDeleter{err: errors.New("db error")})
	t1 := makeTask(courseSessionDeletedPayload{SessionID: uuid.New()})
	if err := w.processTask(context.Background(), t1); err == nil {
		t.Fatal("expected error")
	}
}

func TestCourseSessionDeleted_BadPayload(t *testing.T) {
	w := NewCourseSessionDeletedProcessor(&mockTaskRepo{}, &mockSessionDeleter{})
	if err := w.processTask(context.Background(), domain.Task{Payload: []byte("bad")}); err == nil {
		t.Fatal("expected error")
	}
}

// --- GradingCompletedProcessor ---

type mockLLMGrader struct{ err error }

func (m *mockLLMGrader) ApplyLLMGrade(_ context.Context, _ uuid.UUID, _ int, _ *string) error {
	return m.err
}

func TestGradingCompleted_CharonError_MarksDone(t *testing.T) {
	w := NewGradingCompletedProcessor(&mockTaskRepo{}, &mockLLMGrader{})
	t1 := makeTask(charonGradeResponse{Error: "LLM failed", RequestID: "req-1"})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGradingCompleted_InvalidEvalID(t *testing.T) {
	w := NewGradingCompletedProcessor(&mockTaskRepo{}, &mockLLMGrader{})
	resp := charonGradeResponse{Grades: []struct {
		StudentID string `json:"student_id"`
		Score     int    `json:"score"`
		Comment   string `json:"comment"`
	}{{StudentID: "not-a-uuid", Score: 8}}}
	t1 := makeTask(resp)
	if err := w.processTask(context.Background(), t1); err == nil {
		t.Fatal("expected error")
	}
}

func TestGradingCompleted_ApplyGradeError(t *testing.T) {
	w := NewGradingCompletedProcessor(&mockTaskRepo{}, &mockLLMGrader{err: errors.New("db error")})
	resp := charonGradeResponse{Grades: []struct {
		StudentID string `json:"student_id"`
		Score     int    `json:"score"`
		Comment   string `json:"comment"`
	}{{StudentID: uuid.New().String(), Score: 7}}}
	t1 := makeTask(resp)
	if err := w.processTask(context.Background(), t1); err == nil {
		t.Fatal("expected error")
	}
}

func TestGradingCompleted_Success(t *testing.T) {
	w := NewGradingCompletedProcessor(&mockTaskRepo{}, &mockLLMGrader{})
	resp := charonGradeResponse{Grades: []struct {
		StudentID string `json:"student_id"`
		Score     int    `json:"score"`
		Comment   string `json:"comment"`
	}{{StudentID: uuid.New().String(), Score: 9, Comment: "Хорошо"}}}
	t1 := makeTask(resp)
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGradingCompleted_BadPayload(t *testing.T) {
	w := NewGradingCompletedProcessor(&mockTaskRepo{}, &mockLLMGrader{})
	if err := w.processTask(context.Background(), domain.Task{Payload: []byte("bad")}); err == nil {
		t.Fatal("expected error")
	}
}

// --- GenerationCompletedProcessor ---

type mockGenerationAdder struct{ err error }

func (m *mockGenerationAdder) AddGeneratedQuestions(_ context.Context, _ uuid.UUID, _ []domain.AddQuestionParams) error {
	return m.err
}

type mockQuizGetter struct {
	quiz *domain.QuizTemplate
	err  error
}

func (m *mockQuizGetter) GetByID(_ context.Context, _ uuid.UUID) (*domain.QuizTemplate, error) {
	return m.quiz, m.err
}

func TestGenerationCompleted_EmptyQuestions_MarksDone(t *testing.T) {
	w := NewGenerationCompletedProcessor(&mockTaskRepo{}, &mockGenerationAdder{}, &mockQuizGetter{})
	t1 := makeTask(sphinxCompletedPayload{QuizID: uuid.New(), Questions: nil})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerationCompleted_AddQuestionsError(t *testing.T) {
	adder := &mockGenerationAdder{err: errors.New("db error")}
	w := NewGenerationCompletedProcessor(&mockTaskRepo{}, adder, &mockQuizGetter{})
	t1 := makeTask(sphinxCompletedPayload{
		QuizID: uuid.New(),
		Questions: []sphinxQuestion{
			{Type: "single_choice", Question: "Q?", Answer: "A", Options: []string{"A", "B"}},
		},
	})
	if err := w.processTask(context.Background(), t1); err == nil {
		t.Fatal("expected error")
	}
}

func TestGenerationCompleted_SingleChoice_Success(t *testing.T) {
	quiz := &domain.QuizTemplate{ID: uuid.New(), AuthorID: uuid.New(), Title: "Test"}
	w := NewGenerationCompletedProcessor(&mockTaskRepo{}, &mockGenerationAdder{}, &mockQuizGetter{quiz: quiz})
	t1 := makeTask(sphinxCompletedPayload{
		QuizID: uuid.New(),
		Questions: []sphinxQuestion{
			{Type: "single_choice", Question: "Q?", Answer: "A", Options: []string{"A", "B"}},
		},
	})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerationCompleted_MultipleChoice_Success(t *testing.T) {
	quiz := &domain.QuizTemplate{ID: uuid.New(), AuthorID: uuid.New()}
	w := NewGenerationCompletedProcessor(&mockTaskRepo{}, &mockGenerationAdder{}, &mockQuizGetter{quiz: quiz})
	t1 := makeTask(sphinxCompletedPayload{
		QuizID: uuid.New(),
		Questions: []sphinxQuestion{
			{Type: "multiple_choice", Question: "Q?", Answer: []any{"A", "B"}, Options: []string{"A", "B", "C"}},
		},
	})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerationCompleted_ShortAnswer_Success(t *testing.T) {
	quiz := &domain.QuizTemplate{ID: uuid.New(), AuthorID: uuid.New()}
	w := NewGenerationCompletedProcessor(&mockTaskRepo{}, &mockGenerationAdder{}, &mockQuizGetter{quiz: quiz})
	t1 := makeTask(sphinxCompletedPayload{
		QuizID: uuid.New(),
		Questions: []sphinxQuestion{
			{Type: "short_answer", Question: "Capital?", Answer: "Paris"},
		},
	})
	if err := w.processTask(context.Background(), t1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerationCompleted_UnknownType_Error(t *testing.T) {
	w := NewGenerationCompletedProcessor(&mockTaskRepo{}, &mockGenerationAdder{}, &mockQuizGetter{})
	t1 := makeTask(sphinxCompletedPayload{
		QuizID:    uuid.New(),
		Questions: []sphinxQuestion{{Type: "unknown", Question: "Q?"}},
	})
	if err := w.processTask(context.Background(), t1); err == nil {
		t.Fatal("expected error for unknown question type")
	}
}

func TestGenerationCompleted_BadPayload(t *testing.T) {
	w := NewGenerationCompletedProcessor(&mockTaskRepo{}, &mockGenerationAdder{}, &mockQuizGetter{})
	if err := w.processTask(context.Background(), domain.Task{Payload: []byte("bad")}); err == nil {
		t.Fatal("expected error")
	}
}
