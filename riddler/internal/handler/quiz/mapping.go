package quiz

import (
	"riddler/internal/domain"
	"riddler/internal/transport/dto"
)

func toListItemResponse(item domain.QuizListItem) dto.QuizListItemResponse {
	return dto.QuizListItemResponse{
		ID:          item.ID.String(),
		Title:       item.Title,
		Description: item.Description,
		DefaultSettings: dto.QuizDefaultSettings{
			TotalTimeLimitSec:    item.DefaultSettings.TotalTimeLimitSec,
			QuestionTimeLimitSec: item.DefaultSettings.QuestionTimeLimitSec,
		},
		IsPublic:       item.IsPublic,
		IsDraft:        item.IsDraft,
		NeedEvaluation: item.NeedEvaluation,
		QuestionCount:  item.QuestionCount,
	}
}
