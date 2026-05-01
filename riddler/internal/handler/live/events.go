package live

import "encoding/json"

type wsMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

func encodeMsg(typ string, data any) []byte {
	payload, _ := json.Marshal(data)
	msg, _ := json.Marshal(wsMessage{Type: typ, Data: payload})
	return msg
}

// C→S команды

type cmdKickParticipant struct {
	AttemptID string `json:"attempt_id"`
}

type cmdSubmitAnswer struct {
	QuestionID string         `json:"question_id"`
	AnswerData map[string]any `json:"answer_data"`
}

// S→C события

type evtLobbyParticipant struct {
	AttemptID string  `json:"attempt_id"`
	UserID    *string `json:"user_id,omitempty"`
	Name      *string `json:"name,omitempty"`
}

type evtStateSnapshot struct {
	Phase           string                `json:"phase"`
	Participants    []evtLobbyParticipant `json:"participants,omitempty"`
	CurrentQuestion *evtQuestion          `json:"current_question,omitempty"`
	QuestionIdx     int                   `json:"question_idx,omitempty"`
	TimeLimitSec    int                   `json:"time_limit_sec,omitempty"`
	DeadlineAt      string                `json:"deadline_at,omitempty"`
}

type evtAnswerOption struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	IsCorrect bool   `json:"is_correct,omitempty"`
}

type evtQuestion struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Text     string            `json:"text"`
	ImageID  *string           `json:"image_id,omitempty"`
	MaxScore int               `json:"max_score"`
	Options  []evtAnswerOption `json:"options,omitempty"`
	Metadata map[string]any    `json:"metadata,omitempty"`
}

type evtQuestionStarted struct {
	QuestionIdx  int         `json:"question_idx"`
	Question     evtQuestion `json:"question"`
	TimeLimitSec int         `json:"time_limit_sec"`
	DeadlineAt   string      `json:"deadline_at"`
}

type evtParticipantAnswered struct {
	AttemptID string `json:"attempt_id"`
}

type evtQuestionStatsTick struct {
	AnsweredCount  int `json:"answered_count"`
	ConnectedCount int `json:"connected_count"`
}

type evtOptionStat struct {
	OptionID  string `json:"option_id"`
	Count     int    `json:"count"`
	IsCorrect bool   `json:"is_correct"`
}

type evtMyResult struct {
	IsCorrect bool    `json:"is_correct"`
	Score     float64 `json:"score"`
}

type evtQuestionLocked struct {
	QuestionID   string               `json:"question_id"`
	Stats        evtQuestionStatsTick `json:"stats"`
	Distribution []evtOptionStat      `json:"distribution,omitempty"`
	MyResult     *evtMyResult         `json:"my_result,omitempty"`
}

type evtParticipantKicked struct {
	AttemptID string `json:"attempt_id"`
}
