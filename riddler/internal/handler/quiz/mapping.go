package quiz

import (
	"errors"
	"time"

	"riddler/internal/domain"
	"riddler/internal/transport/dto"
)

var errInvalidMode = errors.New("invalid quiz mode")

func toListItemResponse(item domain.QuizListItem) dto.QuizListItemResponse {
	var attempts []dto.QuizAttemptSummaryResponse
	if len(item.Attempts) > 0 {
		attempts = make([]dto.QuizAttemptSummaryResponse, len(item.Attempts))
		for i, a := range item.Attempts {
			var finishedAt *string
			if a.FinishedAt != nil {
				s := a.FinishedAt.Format(time.RFC3339)
				finishedAt = &s
			}
			attempts[i] = dto.QuizAttemptSummaryResponse{
				ID:          a.ID.String(),
				SessionID:   a.SessionID.String(),
				SessionType: string(a.SessionType),
				Status:      string(a.Status),
				Score:       a.Score,
				StartedAt:   a.StartedAt.Format(time.RFC3339),
				FinishedAt:  finishedAt,
			}
		}
	}
	return dto.QuizListItemResponse{
		ID:              item.ID.String(),
		Title:           item.Title,
		Description:     item.Description,
		DefaultSettings: quizSettingsToDTO(item.DefaultSettings),
		IsPublic:        item.IsPublic,
		Source:          string(item.Source),
		NeedEvaluation:  item.NeedEvaluation,
		QuestionCount:   item.QuestionCount,
		Attempts:        attempts,
	}
}

// quizSettingsToDTO мапит доменные настройки квиза в DTO ответа.
func quizSettingsToDTO(s domain.QuizDefaultSettings) dto.QuizDefaultSettings {
	out := dto.QuizDefaultSettings{
		TotalTimeLimitSec:    s.TotalTimeLimitSec,
		QuestionTimeLimitSec: s.QuestionTimeLimitSec,
		ShuffleQuestions:     s.ShuffleQuestions,
	}
	if s.Mode != nil {
		mode := string(*s.Mode)
		out.Mode = &mode
	}
	if s.StartedAt != nil {
		v := s.StartedAt.Format(time.RFC3339)
		out.StartedAt = &v
	}
	if s.FinishedAt != nil {
		v := s.FinishedAt.Format(time.RFC3339)
		out.FinishedAt = &v
	}
	return out
}

// quizSettingsFromDTO парсит настройки квиза из DTO запроса.
// Валидирует mode (test|live) и даты в формате RFC3339.
func quizSettingsFromDTO(in *dto.QuizDefaultSettings) (domain.QuizDefaultSettings, error) {
	var out domain.QuizDefaultSettings
	if in == nil {
		return out, nil
	}
	out.TotalTimeLimitSec = in.TotalTimeLimitSec
	out.QuestionTimeLimitSec = in.QuestionTimeLimitSec
	out.ShuffleQuestions = in.ShuffleQuestions
	if in.Mode != nil {
		mode := domain.SessionMode(*in.Mode)
		if mode != domain.SessionModeTest && mode != domain.SessionModeLive {
			return out, errInvalidMode
		}
		out.Mode = &mode
	}
	if in.StartedAt != nil {
		t, err := time.Parse(time.RFC3339, *in.StartedAt)
		if err != nil {
			return out, err
		}
		out.StartedAt = &t
	}
	if in.FinishedAt != nil {
		t, err := time.Parse(time.RFC3339, *in.FinishedAt)
		if err != nil {
			return out, err
		}
		out.FinishedAt = &t
	}
	return out, nil
}
