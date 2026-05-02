package live

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

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
	QuestionTotal   int                   `json:"question_total,omitempty"`
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
	QuestionIdx   int         `json:"question_idx"`
	QuestionTotal int         `json:"question_total"`
	Question      evtQuestion `json:"question"`
	TimeLimitSec  int         `json:"time_limit_sec"`
	DeadlineAt    string      `json:"deadline_at"`
}

// participant.answered — учителю (teacher-only)
type evtParticipantAnswered struct {
	AttemptID   string `json:"attempt_id"`
	QuestionID  string `json:"question_id"`
	IsCorrect   bool   `json:"is_correct"`
	TimeTakenMs int64  `json:"time_taken_ms"`
}

type evtOptionStat struct {
	OptionID  string `json:"option_id"`
	Count     int    `json:"count"`
	IsCorrect bool   `json:"is_correct"`
}

// question.stats_tick — агрегаты по текущему вопросу (teacher-only)
type evtQuestionStatsTick struct {
	QuestionID     string `json:"question_id,omitempty"`
	AnsweredCount  int    `json:"answered_count"`
	ConnectedCount int    `json:"connected_count"`
	CorrectCount   int    `json:"correct_count,omitempty"`
	IncorrectCount int    `json:"incorrect_count,omitempty"`
	AvgTimeMs      *int64 `json:"avg_time_ms,omitempty"`
	// "choice" или "binary" — клиент использует для выбора виджета
	Kind         string          `json:"kind,omitempty"`
	Distribution []evtOptionStat `json:"distribution,omitempty"`
}

type evtCorrectAnswer struct {
	CorrectOptionID  *string           `json:"correct_option_id,omitempty"`
	CorrectOptionIDs []string          `json:"correct_option_ids,omitempty"`
	CorrectAnswers   []string          `json:"correct_answers,omitempty"`
	CorrectOrder     []string          `json:"correct_order,omitempty"`
	CorrectPairs     map[string]string `json:"correct_pairs,omitempty"`
}

type evtMyResult struct {
	IsCorrect bool    `json:"is_correct"`
	Score     float64 `json:"score"`
}

type evtQuestionLocked struct {
	QuestionID    string               `json:"question_id"`
	Stats         evtQuestionStatsTick `json:"stats"`
	Distribution  []evtOptionStat      `json:"distribution,omitempty"`
	CorrectAnswer *evtCorrectAnswer    `json:"correct_answer,omitempty"`
	MyResult      *evtMyResult         `json:"my_result,omitempty"`
}

type evtParticipantKicked struct {
	AttemptID string `json:"attempt_id"`
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// buildStats вычисляет агрегированную статистику по вопросу из всех ответов.
// Возвращает stats и distribution (для choice-типов).
func buildStats(q domain.QuestionWithOptions, allAnswers map[uuid.UUID]domain.LiveAnswer, connected int) (evtQuestionStatsTick, []evtOptionStat) {
	answeredCount := len(allAnswers)
	correctN, incorrectN := 0, 0
	var totalTimeMs int64

	for _, ans := range allAnswers {
		if ans.IsCorrect {
			correctN++
		} else {
			incorrectN++
		}
		totalTimeMs += ans.TimeTakenMs
	}

	var avgTimeMs *int64
	if answeredCount > 0 {
		avg := totalTimeMs / int64(answeredCount)
		avgTimeMs = &avg
	}

	isChoice := q.Type == domain.QuestionTypeSingleChoice || q.Type == domain.QuestionTypeMultipleChoice
	kind := "binary"
	var distribution []evtOptionStat

	if isChoice {
		kind = "choice"
		optCount := make(map[string]int)
		for _, ans := range allAnswers {
			if id, ok := ans.AnswerData["selected_option_id"].(string); ok {
				optCount[id]++
			}
			if ids, ok := ans.AnswerData["selected_option_ids"].([]any); ok {
				for _, v := range ids {
					if id, ok := v.(string); ok {
						optCount[id]++
					}
				}
			}
		}
		for _, opt := range q.Options {
			distribution = append(distribution, evtOptionStat{
				OptionID:  opt.ID.String(),
				Count:     optCount[opt.ID.String()],
				IsCorrect: opt.IsCorrect,
			})
		}
	}

	stats := evtQuestionStatsTick{
		QuestionID:     q.ID.String(),
		AnsweredCount:  answeredCount,
		ConnectedCount: connected,
		CorrectCount:   correctN,
		IncorrectCount: incorrectN,
		AvgTimeMs:      avgTimeMs,
		Kind:           kind,
	}
	return stats, distribution
}

// buildCorrectAnswer строит структуру правильного ответа по типу вопроса.
func buildCorrectAnswer(q domain.QuestionWithOptions) *evtCorrectAnswer {
	switch q.Type {
	case domain.QuestionTypeSingleChoice:
		for _, o := range q.Options {
			if o.IsCorrect {
				s := o.ID.String()
				return &evtCorrectAnswer{CorrectOptionID: &s}
			}
		}

	case domain.QuestionTypeMultipleChoice:
		var ids []string
		for _, o := range q.Options {
			if o.IsCorrect {
				ids = append(ids, o.ID.String())
			}
		}
		return &evtCorrectAnswer{CorrectOptionIDs: ids}

	case domain.QuestionTypeWithGivenAnswer:
		if answers, ok := q.Metadata["correct_answers"].([]any); ok {
			strs := make([]string, len(answers))
			for i, a := range answers {
				strs[i] = fmt.Sprintf("%v", a)
			}
			return &evtCorrectAnswer{CorrectAnswers: strs}
		}

	case domain.QuestionTypeDrag:
		if order, ok := q.Metadata["correct_order"].([]any); ok {
			strs := make([]string, len(order))
			for i, o := range order {
				strs[i] = fmt.Sprintf("%v", o)
			}
			return &evtCorrectAnswer{CorrectOrder: strs}
		}

	case domain.QuestionTypeConnection:
		if pairs, ok := q.Metadata["correct_pairs"].(map[string]any); ok {
			m := make(map[string]string, len(pairs))
			for k, v := range pairs {
				m[k] = fmt.Sprintf("%v", v)
			}
			return &evtCorrectAnswer{CorrectPairs: m}
		}
	}
	return nil
}
