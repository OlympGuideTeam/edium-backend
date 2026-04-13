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

type AnswerOptionRequest struct {
	Text      string `json:"text" binding:"required"`
	IsCorrect bool   `json:"is_correct"`
}

type AddQuestionRequest struct {
	Type          string                `json:"type" binding:"required"`
	Text          string                `json:"text" binding:"required"`
	ImageLink     *string               `json:"image_link"`
	Metadata      map[string]any        `json:"metadata"`
	MaxScore      *int                  `json:"max_score"`
	AnswerOptions []AnswerOptionRequest `json:"answer_options" binding:"required"`
}

type AddQuestionResponse struct {
	ID         string `json:"id"`
	OrderIndex int    `json:"order_index"`
}
