package attempt

import (
	"time"

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

func toAttemptResultResponse(r *domain.AttemptResult) dto.AttemptResultResponse {
	answers := make([]dto.AnswerSubmissionResponse, len(r.Answers))
	for i, a := range r.Answers {
		var src *string
		if a.FinalSource != nil {
			s := string(*a.FinalSource)
			src = &s
		}
		answers[i] = dto.AnswerSubmissionResponse{
			QuestionID:    a.QuestionID.String(),
			AnswerData:    a.AnswerData,
			FinalScore:    a.FinalScore,
			FinalSource:   src,
			FinalFeedback: a.FinalFeedback,
		}
	}

	var finishedAt *string
	if r.FinishedAt != nil {
		s := r.FinishedAt.Format(time.RFC3339)
		finishedAt = &s
	}

	return dto.AttemptResultResponse{
		AttemptID:  r.ID.String(),
		Status:     string(r.Status),
		Score:      r.Score,
		StartedAt:  r.StartedAt.Format(time.RFC3339),
		FinishedAt: finishedAt,
		Answers:    answers,
	}
}
