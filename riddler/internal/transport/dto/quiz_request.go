package dto

type AttachToCourseRequest struct {
	CourseID string `json:"course_id" binding:"required"`
}

type CreateQuizRequest struct {
	Title           string                 `json:"title"            binding:"required"`
	Description     *string                `json:"description"`
	DefaultSettings *QuizDefaultSettings   `json:"default_settings"`
	AttachToCourse  *AttachToCourseRequest `json:"attach_to_course"`
}

type UpdateQuizRequest struct {
	Title           *string              `json:"title"`
	Description     *string              `json:"description"`
	DefaultSettings *QuizDefaultSettings `json:"default_settings"`
}

type ReorderQuestionsRequest struct {
	QuestionIDs []string `json:"question_ids" binding:"required,min=1"`
}

type AddQuestionRequest struct {
	Type          string                `json:"type"           binding:"required"`
	Text          string                `json:"text"           binding:"required"`
	ImageID       *string               `json:"image_id"`
	Metadata      map[string]any        `json:"metadata"`
	MaxScore      *int                  `json:"max_score"`
	AnswerOptions []AnswerOptionRequest `json:"answer_options" binding:"required"`
}

type AnswerOptionRequest struct {
	Text      string `json:"text"       binding:"required"`
	IsCorrect bool   `json:"is_correct"`
}

type CreateTestSessionRequest struct {
	QuizTemplateID    string  `json:"quiz_template_id"      binding:"required"`
	ModuleID          string  `json:"module_id"             binding:"required"`
	TotalTimeLimitSec *int    `json:"total_time_limit_sec"`
	ShuffleQuestions  *bool   `json:"shuffle_questions"`
	StartedAt         *string `json:"started_at"`
	FinishedAt        *string `json:"finished_at"`
}

type CopyQuizRequest struct {
	AttachToCourse *AttachToCourseRequest `json:"attach_to_course"`
}

type CreateLiveSessionRequest struct {
	QuizTemplateID       string `json:"quiz_template_id"         binding:"required"`
	ModuleID             string `json:"module_id"                binding:"required"`
	QuestionTimeLimitSec *int   `json:"question_time_limit_sec"`
}

type CreateTestCourseSessionInlineRequest struct {
	Title             string               `json:"title"               binding:"required"`
	Description       *string              `json:"description"`
	CourseID          string               `json:"course_id"           binding:"required"`
	ModuleID          string               `json:"module_id"           binding:"required"`
	Questions         []AddQuestionRequest `json:"questions"           binding:"required,min=1"`
	TotalTimeLimitSec *int                 `json:"total_time_limit_sec"`
	ShuffleQuestions  *bool                `json:"shuffle_questions"`
	StartedAt         *string              `json:"started_at"`
	FinishedAt        *string              `json:"finished_at"`
}

type CreateTestCourseSessionInlineResponse struct {
	QuizTemplateID string `json:"quiz_template_id"`
	SessionID      string `json:"session_id"`
}

type CreateSessionResponse struct {
	SessionID string `json:"session_id"`
}
