package dto

type CreateLiveCourseSessionRequest struct {
	QuizTemplateID       string `json:"quiz_template_id" binding:"required"`
	ModuleID             string `json:"module_id"         binding:"required"`
	QuestionTimeLimitSec *int   `json:"question_time_limit_sec"`
}

type CreateLiveLibrarySessionRequest struct {
	QuizTemplateID       string `json:"quiz_template_id" binding:"required"`
	QuestionTimeLimitSec *int   `json:"question_time_limit_sec"`
}

type CreateLiveSessionResponse struct {
	SessionID string `json:"session_id"`
}

type LiveSessionMetaResponse struct {
	SessionID            string  `json:"session_id"`
	QuizTemplateID       string  `json:"quiz_template_id"`
	QuizTitle            string  `json:"quiz_title"`
	QuestionCount        int     `json:"question_count"`
	Source               string  `json:"source"`
	Phase                string  `json:"phase"`
	JoinCode             *string `json:"join_code,omitempty"`
	QuestionTimeLimitSec int     `json:"question_time_limit_sec"`
	IsAnonymousAllowed   bool    `json:"is_anonymous_allowed"`
	ParticipantsCount    int     `json:"participants_count"`
}

type StartLiveSessionResponse struct {
	WsToken  string `json:"ws_token"`
	JoinCode string `json:"join_code"`
}

type ResolveLiveCodeResponse struct {
	SessionID          string `json:"session_id"`
	QuizTitle          string `json:"quiz_title"`
	QuestionCount      int    `json:"question_count"`
	IsAnonymousAllowed bool   `json:"is_anonymous_allowed"`
}

type JoinLiveSessionRequest struct {
	Name *string `json:"name"`
}

type JoinLiveSessionResponse struct {
	AttemptID string `json:"attempt_id"`
	WsToken   string `json:"ws_token"`
}
