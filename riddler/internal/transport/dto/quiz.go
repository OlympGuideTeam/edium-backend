package dto

type QuizDefaultSettings struct {
	TotalTimeLimitSec    *int `json:"total_time_limit_sec"`
	QuestionTimeLimitSec *int `json:"question_time_limit_sec"`
}

type CreateQuizRequest struct {
	Title           string               `json:"title"             binding:"required"`
	Description     *string              `json:"description"`
	DefaultSettings *QuizDefaultSettings `json:"default_settings"`
}

type CreateQuizResponse struct {
	ID string `json:"id"`
}
