package dto

type QuizDefaultSettings struct {
	TotalTimeLimitSec    *int  `json:"total_time_limit_sec"`
	QuestionTimeLimitSec *int  `json:"question_time_limit_sec"`
	ShuffleQuestions     *bool `json:"shuffle_questions"`
}

type CreateQuizResponse struct {
	ID string `json:"id"`
}

type AddQuestionResponse struct {
	ID         string `json:"id"`
	OrderIndex int    `json:"order_index"`
}

type AnswerOptionResponse struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	IsCorrect bool   `json:"is_correct"`
}

type QuestionResponse struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Text       string                 `json:"text"`
	ImageLink  *string                `json:"image_link"`
	OrderIndex int                    `json:"order_index"`
	MaxScore   int                    `json:"max_score"`
	Metadata   map[string]any         `json:"metadata,omitempty"`
	Options    []AnswerOptionResponse `json:"options"`
}

type QuizDetailResponse struct {
	ID              string              `json:"id"`
	Title           string              `json:"title"`
	Description     *string             `json:"description"`
	DefaultSettings QuizDefaultSettings `json:"default_settings"`
	IsPublic        bool                `json:"is_public"`
	Source          string              `json:"source"`
	NeedEvaluation  bool                `json:"need_evaluation"`
	Questions       []QuestionResponse  `json:"questions"`
}

type QuizStudentDefaultSettings struct {
	TotalTimeLimitSec    *int `json:"total_time_limit_sec"`
	QuestionTimeLimitSec *int `json:"question_time_limit_sec"`
}

type QuizStudentViewResponse struct {
	ID                   string                     `json:"id"`
	Title                string                     `json:"title"`
	Description          *string                    `json:"description"`
	DefaultSettings      QuizStudentDefaultSettings `json:"default_settings"`
	QuestionCount        int                        `json:"question_count"`
	LibraryTestSessionID *string                    `json:"library_test_session_id"`
}

type QuizListItemResponse struct {
	ID              string              `json:"id"`
	Title           string              `json:"title"`
	Description     *string             `json:"description"`
	DefaultSettings QuizDefaultSettings `json:"default_settings"`
	IsPublic        bool                `json:"is_public"`
	Source          string              `json:"source"`
	NeedEvaluation  bool                `json:"need_evaluation"`
	QuestionCount   int                 `json:"question_count"`
}
