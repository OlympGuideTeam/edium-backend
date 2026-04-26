package attempt

import (
	"math/rand/v2"

	"riddler/internal/domain"
)

func sanitizeQuestion(q domain.QuestionWithOptions) domain.QuestionForStudent {
	s := domain.QuestionForStudent{
		ID:       q.ID,
		Type:     q.Type,
		Text:     q.Text,
		ImageID:  q.ImageID,
		MaxScore: q.MaxScore,
	}

	switch q.Type {
	case domain.QuestionTypeSingleChoice, domain.QuestionTypeMultipleChoice:
		s.Options = make([]domain.AnswerOptionForStudent, len(q.Options))
		for i, o := range q.Options {
			s.Options[i] = domain.AnswerOptionForStudent{ID: o.ID, Text: o.Text}
		}

	case domain.QuestionTypeWithGivenAnswer, domain.QuestionTypeWithFreeAnswer:
		// метаданные не нужны студенту

	case domain.QuestionTypeDrag:
		if order, ok := q.Metadata["correct_order"].([]any); ok {
			items := make([]any, len(order))
			copy(items, order)
			rand.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })
			s.Metadata = map[string]any{"items": items}
		}

	case domain.QuestionTypeConnection:
		meta := make(map[string]any)
		if left, ok := q.Metadata["left"]; ok {
			meta["left"] = left
		}
		if right, ok := q.Metadata["right"].([]any); ok {
			shuffled := make([]any, len(right))
			copy(shuffled, right)
			rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
			meta["right"] = shuffled
		}
		s.Metadata = meta
	}

	return s
}
