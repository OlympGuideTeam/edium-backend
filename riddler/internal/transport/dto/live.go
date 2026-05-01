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

type LiveLeaderboardRow struct {
	Position  int     `json:"position"`
	AttemptID string  `json:"attempt_id"`
	UserID    *string `json:"user_id,omitempty"`
	Name      *string `json:"name,omitempty"`
	Score     float64 `json:"score"`
	IsMe      bool    `json:"is_me,omitempty"`
}

type LiveResultsStudentResponse struct {
	MyPosition        int                  `json:"my_position"`
	TotalParticipants int                  `json:"total_participants"`
	MyScore           float64              `json:"my_score"`
	MaxScore          float64              `json:"max_score"`
	CorrectCount      int                  `json:"correct_count"`
	QuestionsCount    int                  `json:"questions_count"`
	Top               []LiveLeaderboardRow `json:"top"`
}

type LiveOptionStat struct {
	OptionID  string `json:"option_id"`
	Count     int    `json:"count"`
	IsCorrect bool   `json:"is_correct"`
}

type LiveQuestionResult struct {
	QuestionID    string           `json:"question_id"`
	OrderIndex    int              `json:"order_index"`
	Text          string           `json:"text"`
	Type          string           `json:"type"`
	CorrectRate   float64          `json:"correct_rate"`
	AnsweredCount int              `json:"answered_count"`
	CorrectCount  int              `json:"correct_count"`
	AvgTimesMs    *int             `json:"avg_time_ms,omitempty"`
	Distribution  []LiveOptionStat `json:"distribution,omitempty"`
}

type LiveResultsTeacherAttempt struct {
	Position     int                  `json:"position"`
	AttemptID    string               `json:"attempt_id"`
	UserID       *string              `json:"user_id,omitempty"`
	Name         *string              `json:"name,omitempty"`
	Score        float64              `json:"score"`
	CorrectCount int                  `json:"correct_count"`
}

type LiveResultsTeacherResponse struct {
	Questions   []LiveQuestionResult        `json:"questions"`
	Leaderboard []LiveResultsTeacherAttempt `json:"leaderboard"`
}

type JoinLiveSessionRequest struct {
	Name *string `json:"name"`
}

type JoinLiveSessionResponse struct {
	AttemptID string `json:"attempt_id"`
	WsToken   string `json:"ws_token"`
}
