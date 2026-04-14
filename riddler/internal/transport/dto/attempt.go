package dto

// --- Запросы ---

type SubmitAnswerRequest struct {
	QuestionID string         `json:"question_id" binding:"required"`
	AnswerData map[string]any `json:"answer_data" binding:"required"`
}

// --- Ответы ---

type AnswerOptionForStudentResponse struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type QuestionForStudentResponse struct {
	ID        string                           `json:"id"`
	Type      string                           `json:"type"`
	Text      string                           `json:"text"`
	ImageLink *string                          `json:"image_link,omitempty"`
	MaxScore  int                              `json:"max_score"`
	Options   []AnswerOptionForStudentResponse `json:"options,omitempty"`
	Metadata  map[string]any                   `json:"metadata,omitempty"`
}

type CreateAttemptResponse struct {
	AttemptID string                       `json:"attempt_id"`
	Questions []QuestionForStudentResponse `json:"questions"`
}

type AnswerSubmissionResponse struct {
	QuestionID    string         `json:"question_id"`
	AnswerData    map[string]any `json:"answer_data"`
	FinalScore    *float64       `json:"final_score"`
	FinalSource   *string        `json:"final_source"`
	FinalFeedback *string        `json:"final_feedback"`
}

type AttemptResultResponse struct {
	AttemptID  string                     `json:"attempt_id"`
	Status     string                     `json:"status"`
	Score      *float64                   `json:"score"`
	StartedAt  string                     `json:"started_at"`
	FinishedAt *string                    `json:"finished_at"`
	Answers    []AnswerSubmissionResponse `json:"answers"`
}
