package dto

type CreateQuizRequest struct {
	Title           string               `json:"title"             binding:"required"`
	Description     *string              `json:"description"`
	DefaultSettings *QuizDefaultSettings `json:"default_settings"`
}

type UpdateQuizRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

type PublishQuizRequest struct {
	IsPublic bool `json:"is_public"`
}

type ReorderQuestionsRequest struct {
	QuestionIDs []string `json:"question_ids" binding:"required,min=1"`
}

type AddQuestionRequest struct {
	Type          string                `json:"type"           binding:"required"`
	Text          string                `json:"text"           binding:"required"`
	ImageLink     *string               `json:"image_link"`
	Metadata      map[string]any        `json:"metadata"`
	MaxScore      *int                  `json:"max_score"`
	AnswerOptions []AnswerOptionRequest `json:"answer_options" binding:"required"`
}

type AnswerOptionRequest struct {
	Text      string `json:"text"       binding:"required"`
	IsCorrect bool   `json:"is_correct"`
}
