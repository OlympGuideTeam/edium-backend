package attempt

import (
	"math"

	"riddler/internal/domain"
)

// gradeAnswer возвращает набранный балл за ответ.
// with_free_answer не оценивается автоматически — всегда 0.
func gradeAnswer(q domain.QuestionWithOptions, answerData map[string]any) float64 {
	switch q.Type {
	case domain.QuestionTypeSingleChoice:
		return gradeSingleChoice(q, answerData)
	case domain.QuestionTypeMultipleChoice:
		return gradeMultipleChoice(q, answerData)
	case domain.QuestionTypeWithGivenAnswer:
		return gradeWithGivenAnswer(q, answerData)
	case domain.QuestionTypeDrag:
		return gradeDrag(q, answerData)
	case domain.QuestionTypeConnection:
		return gradeConnection(q, answerData)
	default:
		return 0
	}
}

func gradeSingleChoice(q domain.QuestionWithOptions, data map[string]any) float64 {
	selectedID, _ := data["selected_option_id"].(string)
	for _, o := range q.Options {
		if o.ID.String() == selectedID && o.IsCorrect {
			return float64(q.MaxScore)
		}
	}
	return 0
}

// gradeMultipleChoice даёт частичные баллы:
// score = max(0, (правильно_выбрано - неправильно_выбрано) / всего_правильных) * max_score
func gradeMultipleChoice(q domain.QuestionWithOptions, data map[string]any) float64 {
	raw, _ := data["selected_option_ids"].([]any)
	selected := make(map[string]bool, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			selected[s] = true
		}
	}

	correct := make(map[string]bool)
	for _, o := range q.Options {
		if o.IsCorrect {
			correct[o.ID.String()] = true
		}
	}
	if len(correct) == 0 {
		return 0
	}

	var correctSelected, wrongSelected int
	for id := range selected {
		if correct[id] {
			correctSelected++
		} else {
			wrongSelected++
		}
	}

	ratio := float64(correctSelected-wrongSelected) / float64(len(correct))
	if ratio < 0 {
		ratio = 0
	}
	return math.Round(ratio*float64(q.MaxScore)*100) / 100
}

func gradeWithGivenAnswer(q domain.QuestionWithOptions, data map[string]any) float64 {
	text, _ := data["text"].(string)
	correctAnswers, _ := q.Metadata["correct_answers"].([]any)
	for _, ca := range correctAnswers {
		if s, ok := ca.(string); ok && s == text {
			return float64(q.MaxScore)
		}
	}
	return 0
}

func gradeDrag(q domain.QuestionWithOptions, data map[string]any) float64 {
	submitted, _ := data["order"].([]any)
	correctOrder, _ := q.Metadata["correct_order"].([]any)
	if len(submitted) != len(correctOrder) {
		return 0
	}
	for i, v := range correctOrder {
		if submitted[i] != v {
			return 0
		}
	}
	return float64(q.MaxScore)
}

func gradeConnection(q domain.QuestionWithOptions, data map[string]any) float64 {
	submitted, _ := data["pairs"].(map[string]any)
	correctPairs, _ := q.Metadata["correct_pairs"].(map[string]any)
	if len(submitted) != len(correctPairs) {
		return 0
	}
	for k, v := range correctPairs {
		if submitted[k] != v {
			return 0
		}
	}
	return float64(q.MaxScore)
}
