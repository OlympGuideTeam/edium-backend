package grading

import (
	"testing"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

func optionID() uuid.UUID { return uuid.New() }

func makeOption(id uuid.UUID, text string, correct bool) domain.AnswerOption {
	return domain.AnswerOption{ID: id, Text: text, IsCorrect: correct}
}

// --- IsCorrect ---

func TestIsCorrect(t *testing.T) {
	if !IsCorrect(10, 10) {
		t.Error("10/10 should be correct")
	}
	if IsCorrect(9, 10) {
		t.Error("9/10 should not be correct")
	}
	if IsCorrect(5, 0) {
		t.Error("maxScore=0 should not be correct")
	}
}

// --- ComputeGrade ---

func TestComputeGrade(t *testing.T) {
	if ComputeGrade(10, 10) != 10.0 {
		t.Error("expected 10.0")
	}
	if ComputeGrade(5, 10) != 5.0 {
		t.Error("expected 5.0")
	}
	if ComputeGrade(0, 0) != 0 {
		t.Error("maxScore=0 should return 0")
	}
	if ComputeGrade(7, 10) != 7.0 {
		t.Errorf("expected 7.0, got %v", ComputeGrade(7, 10))
	}
}

// --- single_choice ---

func TestGradeAnswer_SingleChoice_Correct(t *testing.T) {
	id := optionID()
	q := domain.QuestionWithOptions{
		Question: domain.Question{Type: domain.QuestionTypeSingleChoice, MaxScore: 10},
		Options:  []domain.AnswerOption{makeOption(id, "A", true), makeOption(optionID(), "B", false)},
	}
	score := GradeAnswer(q, map[string]any{"selected_option_id": id.String()})
	if score != 10 {
		t.Errorf("expected 10, got %v", score)
	}
}

func TestGradeAnswer_SingleChoice_Wrong(t *testing.T) {
	q := domain.QuestionWithOptions{
		Question: domain.Question{Type: domain.QuestionTypeSingleChoice, MaxScore: 10},
		Options: []domain.AnswerOption{
			makeOption(optionID(), "A", true),
			makeOption(optionID(), "B", false),
		},
	}
	score := GradeAnswer(q, map[string]any{"selected_option_id": uuid.New().String()})
	if score != 0 {
		t.Errorf("expected 0, got %v", score)
	}
}

func TestGradeAnswer_SingleChoice_NoAnswer(t *testing.T) {
	q := domain.QuestionWithOptions{
		Question: domain.Question{Type: domain.QuestionTypeSingleChoice, MaxScore: 10},
		Options:  []domain.AnswerOption{makeOption(optionID(), "A", true)},
	}
	if GradeAnswer(q, map[string]any{}) != 0 {
		t.Error("expected 0 for no answer")
	}
}

// --- multiple_choice ---

func TestGradeAnswer_MultipleChoice_AllCorrect(t *testing.T) {
	id1, id2 := optionID(), optionID()
	q := domain.QuestionWithOptions{
		Question: domain.Question{Type: domain.QuestionTypeMultipleChoice, MaxScore: 10},
		Options: []domain.AnswerOption{
			makeOption(id1, "A", true),
			makeOption(id2, "B", true),
			makeOption(optionID(), "C", false),
		},
	}
	score := GradeAnswer(q, map[string]any{"selected_option_ids": []any{id1.String(), id2.String()}})
	if score != 10 {
		t.Errorf("expected 10, got %v", score)
	}
}

func TestGradeAnswer_MultipleChoice_OneCorrectOneWrong(t *testing.T) {
	id1 := optionID()
	wrongID := optionID()
	q := domain.QuestionWithOptions{
		Question: domain.Question{Type: domain.QuestionTypeMultipleChoice, MaxScore: 10},
		Options: []domain.AnswerOption{
			makeOption(id1, "A", true),
			makeOption(optionID(), "B", true),
			makeOption(wrongID, "C", false),
		},
	}
	score := GradeAnswer(q, map[string]any{"selected_option_ids": []any{id1.String(), wrongID.String()}})
	if score < 0 || score >= 10 {
		t.Errorf("expected partial score, got %v", score)
	}
}

func TestGradeAnswer_MultipleChoice_AllWrong(t *testing.T) {
	wrongID := optionID()
	q := domain.QuestionWithOptions{
		Question: domain.Question{Type: domain.QuestionTypeMultipleChoice, MaxScore: 10},
		Options: []domain.AnswerOption{
			makeOption(optionID(), "A", true),
			makeOption(wrongID, "B", false),
		},
	}
	score := GradeAnswer(q, map[string]any{"selected_option_ids": []any{wrongID.String()}})
	if score != 0 {
		t.Errorf("expected 0, got %v", score)
	}
}

func TestGradeAnswer_MultipleChoice_NoOptions(t *testing.T) {
	q := domain.QuestionWithOptions{
		Question: domain.Question{Type: domain.QuestionTypeMultipleChoice, MaxScore: 10},
	}
	if GradeAnswer(q, map[string]any{}) != 0 {
		t.Error("expected 0 when no correct options")
	}
}

// --- with_given_answer ---

func TestGradeAnswer_WithGivenAnswer_Correct(t *testing.T) {
	q := domain.QuestionWithOptions{
		Question: domain.Question{
			Type:     domain.QuestionTypeWithGivenAnswer,
			MaxScore: 10,
			Metadata: map[string]any{"correct_answers": []any{"Paris", "paris"}},
		},
	}
	if GradeAnswer(q, map[string]any{"text": "Paris"}) != 10 {
		t.Error("expected 10")
	}
	if GradeAnswer(q, map[string]any{"text": "PARIS"}) != 10 {
		t.Error("expected 10 (case-insensitive)")
	}
}

func TestGradeAnswer_WithGivenAnswer_Wrong(t *testing.T) {
	q := domain.QuestionWithOptions{
		Question: domain.Question{
			Type:     domain.QuestionTypeWithGivenAnswer,
			MaxScore: 10,
			Metadata: map[string]any{"correct_answers": []any{"Paris"}},
		},
	}
	if GradeAnswer(q, map[string]any{"text": "London"}) != 0 {
		t.Error("expected 0")
	}
}

func TestGradeAnswer_WithGivenAnswer_TrimSpace(t *testing.T) {
	q := domain.QuestionWithOptions{
		Question: domain.Question{
			Type:     domain.QuestionTypeWithGivenAnswer,
			MaxScore: 5,
			Metadata: map[string]any{"correct_answers": []any{"Paris"}},
		},
	}
	if GradeAnswer(q, map[string]any{"text": "  Paris  "}) != 5 {
		t.Error("expected 5 with trimmed answer")
	}
}

// --- drag ---

func TestGradeAnswer_Drag_Correct(t *testing.T) {
	q := domain.QuestionWithOptions{
		Question: domain.Question{
			Type:     domain.QuestionTypeDrag,
			MaxScore: 10,
			Metadata: map[string]any{"correct_order": []any{"A", "B", "C"}},
		},
	}
	if GradeAnswer(q, map[string]any{"order": []any{"A", "B", "C"}}) != 10 {
		t.Error("expected 10")
	}
}

func TestGradeAnswer_Drag_Wrong(t *testing.T) {
	q := domain.QuestionWithOptions{
		Question: domain.Question{
			Type:     domain.QuestionTypeDrag,
			MaxScore: 10,
			Metadata: map[string]any{"correct_order": []any{"A", "B", "C"}},
		},
	}
	if GradeAnswer(q, map[string]any{"order": []any{"B", "A", "C"}}) != 0 {
		t.Error("expected 0")
	}
}

func TestGradeAnswer_Drag_WrongLength(t *testing.T) {
	q := domain.QuestionWithOptions{
		Question: domain.Question{
			Type:     domain.QuestionTypeDrag,
			MaxScore: 10,
			Metadata: map[string]any{"correct_order": []any{"A", "B"}},
		},
	}
	if GradeAnswer(q, map[string]any{"order": []any{"A"}}) != 0 {
		t.Error("expected 0 for wrong length")
	}
}

// --- connection ---

func TestGradeAnswer_Connection_Correct(t *testing.T) {
	q := domain.QuestionWithOptions{
		Question: domain.Question{
			Type:     domain.QuestionTypeConnection,
			MaxScore: 10,
			Metadata: map[string]any{"correct_pairs": map[string]any{"1": "A", "2": "B"}},
		},
	}
	if GradeAnswer(q, map[string]any{"pairs": map[string]any{"1": "A", "2": "B"}}) != 10 {
		t.Error("expected 10")
	}
}

func TestGradeAnswer_Connection_Wrong(t *testing.T) {
	q := domain.QuestionWithOptions{
		Question: domain.Question{
			Type:     domain.QuestionTypeConnection,
			MaxScore: 10,
			Metadata: map[string]any{"correct_pairs": map[string]any{"1": "A", "2": "B"}},
		},
	}
	if GradeAnswer(q, map[string]any{"pairs": map[string]any{"1": "B", "2": "A"}}) != 0 {
		t.Error("expected 0")
	}
}

func TestGradeAnswer_Connection_WrongSize(t *testing.T) {
	q := domain.QuestionWithOptions{
		Question: domain.Question{
			Type:     domain.QuestionTypeConnection,
			MaxScore: 10,
			Metadata: map[string]any{"correct_pairs": map[string]any{"1": "A", "2": "B"}},
		},
	}
	if GradeAnswer(q, map[string]any{"pairs": map[string]any{"1": "A"}}) != 0 {
		t.Error("expected 0 for wrong size")
	}
}

// --- free_answer (always 0 from auto-grader) ---

func TestGradeAnswer_FreeAnswer(t *testing.T) {
	q := domain.QuestionWithOptions{
		Question: domain.Question{Type: domain.QuestionTypeWithFreeAnswer, MaxScore: 10},
	}
	if GradeAnswer(q, map[string]any{"text": "some answer"}) != 0 {
		t.Error("expected 0 for free answer (manual grading)")
	}
}

// --- unknown type ---

func TestGradeAnswer_UnknownType(t *testing.T) {
	q := domain.QuestionWithOptions{
		Question: domain.Question{Type: "unknown", MaxScore: 10},
	}
	if GradeAnswer(q, map[string]any{}) != 0 {
		t.Error("expected 0 for unknown type")
	}
}
