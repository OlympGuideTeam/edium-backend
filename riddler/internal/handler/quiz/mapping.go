package quiz

import (
	"errors"
	"time"

	"riddler/internal/domain"
	"riddler/internal/transport/dto"
)

var errInvalidMode = errors.New("invalid quiz mode")

func toListItemResponse(item domain.QuizListItem) dto.QuizListItemResponse {
	return dto.QuizListItemResponse{
		ID:              item.ID.String(),
		Title:           item.Title,
		Description:     item.Description,
		DefaultSettings: quizSettingsToDTO(item.DefaultSettings),
		IsPublic:        item.IsPublic,
		Source:          string(item.Source),
		NeedEvaluation:  item.NeedEvaluation,
		QuestionCount:   item.QuestionCount,
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
