package quiz

import (
	"errors"
	"testing"

	"riddler/internal/domain"
	"riddler/internal/pkg/apperr"
)

func twoOptions(correct int) []domain.AddOptionParams {
	opts := []domain.AddOptionParams{
		{Text: "A", IsCorrect: false},
		{Text: "B", IsCorrect: false},
	}
	if correct >= 0 && correct < len(opts) {
		opts[correct].IsCorrect = true
	}
	return opts
}

// --- single_choice ---

func TestValidate_SingleChoice_OK(t *testing.T) {
	if err := validateQuestion(domain.QuestionTypeSingleChoice, nil, twoOptions(0)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_SingleChoice_NoOptions(t *testing.T) {
	err := validateQuestion(domain.QuestionTypeSingleChoice, nil, nil)
	if !errors.Is(err, apperr.ErrQuestionOptionsRequired) {
		t.Fatalf("expected ErrQuestionOptionsRequired, got %v", err)
	}
}

func TestValidate_SingleChoice_OneOption(t *testing.T) {
	err := validateQuestion(domain.QuestionTypeSingleChoice, nil, []domain.AddOptionParams{{Text: "A", IsCorrect: true}})
	if !errors.Is(err, apperr.ErrQuestionOptionsRequired) {
		t.Fatalf("expected ErrQuestionOptionsRequired, got %v", err)
	}
}

func TestValidate_SingleChoice_NoCorrect(t *testing.T) {
	opts := []domain.AddOptionParams{{Text: "A"}, {Text: "B"}}
	err := validateQuestion(domain.QuestionTypeSingleChoice, nil, opts)
	if !errors.Is(err, apperr.ErrQuestionOneCorrect) {
		t.Fatalf("expected ErrQuestionOneCorrect, got %v", err)
	}
}

func TestValidate_SingleChoice_TwoCorrect(t *testing.T) {
	opts := []domain.AddOptionParams{{Text: "A", IsCorrect: true}, {Text: "B", IsCorrect: true}}
	err := validateQuestion(domain.QuestionTypeSingleChoice, nil, opts)
	if !errors.Is(err, apperr.ErrQuestionOneCorrect) {
		t.Fatalf("expected ErrQuestionOneCorrect, got %v", err)
	}
}

// --- multiple_choice ---

func TestValidate_MultipleChoice_OK(t *testing.T) {
	if err := validateQuestion(domain.QuestionTypeMultipleChoice, nil, twoOptions(0)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_MultipleChoice_NoCorrect(t *testing.T) {
	opts := []domain.AddOptionParams{{Text: "A"}, {Text: "B"}}
	err := validateQuestion(domain.QuestionTypeMultipleChoice, nil, opts)
	if !errors.Is(err, apperr.ErrQuestionNoCorrect) {
		t.Fatalf("expected ErrQuestionNoCorrect, got %v", err)
	}
}

// --- with_given_answer ---

func TestValidate_WithGivenAnswer_OK(t *testing.T) {
	meta := map[string]any{"correct_answers": []any{"Paris"}}
	if err := validateQuestion(domain.QuestionTypeWithGivenAnswer, meta, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_WithGivenAnswer_NoMeta(t *testing.T) {
	err := validateQuestion(domain.QuestionTypeWithGivenAnswer, nil, nil)
	if !errors.Is(err, apperr.ErrQuestionMetadataInvalid) {
		t.Fatalf("expected ErrQuestionMetadataInvalid, got %v", err)
	}
}

func TestValidate_WithGivenAnswer_EmptyAnswers(t *testing.T) {
	meta := map[string]any{"correct_answers": []any{}}
	err := validateQuestion(domain.QuestionTypeWithGivenAnswer, meta, nil)
	if !errors.Is(err, apperr.ErrQuestionMetadataInvalid) {
		t.Fatalf("expected ErrQuestionMetadataInvalid, got %v", err)
	}
}

// --- with_free_answer ---

func TestValidate_FreeAnswer_OK(t *testing.T) {
	if err := validateQuestion(domain.QuestionTypeWithFreeAnswer, nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- drag ---

func TestValidate_Drag_OK(t *testing.T) {
	meta := map[string]any{"correct_order": []any{"A", "B", "C"}}
	if err := validateQuestion(domain.QuestionTypeDrag, meta, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_Drag_NoMeta(t *testing.T) {
	err := validateQuestion(domain.QuestionTypeDrag, nil, nil)
	if !errors.Is(err, apperr.ErrQuestionMetadataInvalid) {
		t.Fatalf("expected ErrQuestionMetadataInvalid, got %v", err)
	}
}

// --- connection ---

func TestValidate_Connection_OK(t *testing.T) {
	meta := map[string]any{
		"left":          []any{"A"},
		"right":         []any{"1"},
		"correct_pairs": map[string]any{"A": "1"},
	}
	if err := validateQuestion(domain.QuestionTypeConnection, meta, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_Connection_MissingLeft(t *testing.T) {
	meta := map[string]any{
		"right":         []any{"1"},
		"correct_pairs": map[string]any{"A": "1"},
	}
	err := validateQuestion(domain.QuestionTypeConnection, meta, nil)
	if !errors.Is(err, apperr.ErrQuestionMetadataInvalid) {
		t.Fatalf("expected ErrQuestionMetadataInvalid, got %v", err)
	}
}

func TestValidate_Connection_MissingPairs(t *testing.T) {
	meta := map[string]any{
		"left":  []any{"A"},
		"right": []any{"1"},
	}
	err := validateQuestion(domain.QuestionTypeConnection, meta, nil)
	if !errors.Is(err, apperr.ErrQuestionMetadataInvalid) {
		t.Fatalf("expected ErrQuestionMetadataInvalid, got %v", err)
	}
}

// --- unknown type ---

func TestValidate_UnknownType(t *testing.T) {
	err := validateQuestion("unknown_type", nil, nil)
	if !errors.Is(err, apperr.ErrQuestionInvalidType) {
		t.Fatalf("expected ErrQuestionInvalidType, got %v", err)
	}
}
