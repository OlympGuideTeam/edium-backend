package dto

// --- Запросы ---

type SubmitAnswerRequest struct {
	QuestionID string         `json:"question_id" binding:"required"`
	AnswerData map[string]any `json:"answer_data" binding:"required"`
}

type GradeItem struct {
	SubmissionID string  `json:"submission_id" binding:"required"`
	Score        float64 `json:"score" binding:"min=0"`
	Feedback     *string `json:"feedback"`
}

type GradeAttemptRequest struct {
	Grades []GradeItem `json:"grades" binding:"required,min=1"`
}

// --- Ответы ---

type AnswerOptionForStudentResponse struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type QuestionForStudentResponse struct {
	ID       string                           `json:"id"`
	Type     string                           `json:"type"`
	Text     string                           `json:"text"`
	ImageID  *string                          `json:"image_id,omitempty"`
	MaxScore int                              `json:"max_score"`
	Options  []AnswerOptionForStudentResponse `json:"options,omitempty"`
	Metadata map[string]any                   `json:"metadata,omitempty"`
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

type AttemptSummaryResponse struct {
	AttemptID string   `json:"attempt_id"`
	UserID    string   `json:"user_id,omitempty"`
	Status    string   `json:"status"`
	Score     *float64 `json:"score"`
}

type AnswerOptionTeacherResponse struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	IsCorrect bool   `json:"is_correct"`
}

type AnswerReviewResponse struct {
	SubmissionID   string                           `json:"submission_id"`
	QuestionID     string                           `json:"question_id"`
	QuestionType   string                           `json:"question_type"`
	QuestionText   string                           `json:"question_text"`
	AnswerData     map[string]any                   `json:"answer_data"`
	FinalScore     *float64                         `json:"final_score"`
	FinalSource    *string                          `json:"final_source"`
	FinalFeedback  *string                          `json:"final_feedback"`
	Options        []AnswerOptionTeacherResponse    `json:"options,omitempty"`
	StudentOptions []AnswerOptionForStudentResponse `json:"student_options,omitempty"`
	Metadata       map[string]any                   `json:"metadata,omitempty"`
}

type AttemptReviewResponse struct {
	AttemptID  string                 `json:"attempt_id"`
	UserID     string                 `json:"user_id,omitempty"`
	Status     string                 `json:"status"`
	Score      *float64               `json:"score"`
	StartedAt  string                 `json:"started_at"`
	FinishedAt *string                `json:"finished_at"`
	Answers    []AnswerReviewResponse `json:"answers"`
}

type RecentGradeItemDTO struct {
	SessionID      string   `json:"session_id"`
	QuizTemplateID string   `json:"quiz_template_id"`
	QuizTitle      string   `json:"quiz_title"`
	AttemptID      string   `json:"attempt_id"`
	Score          *float64 `json:"score"`
	Status         string   `json:"status"`
	FinishedAt     *string  `json:"finished_at"`
}

type ActiveTestItemDTO struct {
	SessionID         string   `json:"session_id"`
	QuizTemplateID    string   `json:"quiz_template_id"`
	QuizTitle         string   `json:"quiz_title"`
	TotalTimeLimitSec *int     `json:"total_time_limit_sec"`
	SessionStartedAt  *string  `json:"session_started_at"`
	SessionFinishedAt *string  `json:"session_finished_at"`
	AttemptID         *string  `json:"attempt_id"`
	AttemptStatus     *string  `json:"attempt_status"`
}

type StudentDashboardResponse struct {
	RecentGrades []RecentGradeItemDTO `json:"recent_grades"`
	ActiveTests  []ActiveTestItemDTO  `json:"active_tests"`
}

type AwaitingReviewSessionItem struct {
	SessionID      string `json:"session_id"`
	QuizTemplateID string `json:"quiz_template_id"`
	QuizTitle      string `json:"quiz_title"`
	GradingCount   int    `json:"grading_count"`
	GradedCount    int    `json:"graded_count"`
	CompletedCount int    `json:"completed_count"`
}

type AwaitingReviewResponse struct {
	Sessions []AwaitingReviewSessionItem `json:"sessions"`
}

type UserStatisticResponse struct {
	QuizCountPassed       int     `json:"quiz_count_passed"`
	AvgQuizScore          float64 `json:"avg_quiz_score"`
	QuizSessionsConducted int     `json:"quiz_sessions_conducted"`
}

type AttemptResultResponse struct {
	AttemptID  string                     `json:"attempt_id"`
	Status     string                     `json:"status"`
	Score      *float64                   `json:"score"`
	StartedAt  string                     `json:"started_at"`
	FinishedAt *string                    `json:"finished_at"`
	Answers    []AnswerSubmissionResponse `json:"answers"`
}
