package attempt

import (
	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/transport/dto"
)

func toQuestionsResponse(questions []domain.QuestionForStudent) []dto.QuestionForStudentResponse {
	out := make([]dto.QuestionForStudentResponse, len(questions))
	for i, q := range questions {
		var options []dto.AnswerOptionForStudentResponse
		if len(q.Options) > 0 {
			options = make([]dto.AnswerOptionForStudentResponse, len(q.Options))
			for j, o := range q.Options {
				options[j] = dto.AnswerOptionForStudentResponse{
					ID:   o.ID.String(),
					Text: o.Text,
				}
			}
		}
		out[i] = dto.QuestionForStudentResponse{
			ID:       q.ID.String(),
			Type:     string(q.Type),
			Text:     q.Text,
			ImageID:  uuidPtrToString(q.ImageID),
			MaxScore: q.MaxScore,
			Options:  options,
			Metadata: q.Metadata,
		}
	}
	return out
}

// sanitizeMetadataForStudent strips correct-answer fields from question metadata,
// leaving only the items/sides the student needs to understand the question.
func sanitizeMetadataForStudent(questionType string, metadata map[string]any) map[string]any {
	if metadata == nil {
		return nil
	}
	switch domain.QuestionType(questionType) {
	case domain.QuestionTypeDrag:
		if order, ok := metadata["correct_order"].([]any); ok {
			return map[string]any{"items": order}
		}
		return nil
	case domain.QuestionTypeConnection:
		out := make(map[string]any, 2)
		if left, ok := metadata["left"]; ok {
			out["left"] = left
		}
		if right, ok := metadata["right"]; ok {
			out["right"] = right
		}
		if len(out) == 0 {
			return nil
		}
		return out
	}
	return nil
}

func uuidPtrToString(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}
