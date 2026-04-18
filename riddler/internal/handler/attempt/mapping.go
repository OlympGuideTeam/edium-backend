package attempt

import (
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
			ID:        q.ID.String(),
			Type:      string(q.Type),
			Text:      q.Text,
			ImageLink: q.ImageLink,
			MaxScore:  q.MaxScore,
			Options:   options,
			Metadata:  q.Metadata,
		}
	}
	return out
}
