package live

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/pkg/grading"
)

func (h *Handler) handleStudentCmd(ctx context.Context, conn *Conn, room *SessionRoom, sessionID uuid.UUID, msg wsMessage) {
	if msg.Type != "student.submit_answer" {
		return
	}
	var cmd cmdSubmitAnswer
	if err := json.Unmarshal(msg.Data, &cmd); err != nil {
		return
	}
	questionID, err := uuid.Parse(cmd.QuestionID)
	if err != nil {
		return
	}
	h.handleSubmitAnswer(ctx, conn, room, sessionID, questionID, cmd.AnswerData)
}

func (h *Handler) handleSubmitAnswer(ctx context.Context, conn *Conn, room *SessionRoom, sessionID, questionID uuid.UUID, answerData map[string]any) {
	phase, err := h.svc.GetPhase(ctx, sessionID)
	if err != nil || phase != domain.LivePhaseQuestionActive {
		return
	}

	room.mu.RLock()
	idx := room.questionIdx
	questions := room.questions
	startedAt := room.questionStartedAt
	room.mu.RUnlock()

	if idx >= len(questions) || questions[idx].ID != questionID {
		return
	}

	existing, err := h.svc.GetAnswer(ctx, sessionID, questionID, conn.attemptID)
	if err != nil || existing != nil {
		return
	}

	q := questions[idx]
	score := grading.GradeAnswer(q, answerData)
	isCorrect := grading.IsCorrect(score, float64(q.MaxScore))
	timeTakenMs := time.Since(startedAt).Milliseconds()

	ans := domain.LiveAnswer{
		AnswerData:  answerData,
		IsCorrect:   isCorrect,
		Score:       score,
		TimeTakenMs: timeTakenMs,
	}
	if err := h.svc.SaveAnswer(ctx, sessionID, questionID, conn.attemptID, ans); err != nil {
		return
	}

	room.sendTeacher(encodeMsg("participant.answered", evtParticipantAnswered{
		AttemptID: conn.attemptID.String(),
	}))

	h.sendStatsTick(ctx, room, sessionID, questionID)

	answered, _ := h.svc.GetAnsweredCount(ctx, sessionID, questionID)
	connected := room.connectedCount()
	if connected > 0 && answered >= connected {
		h.lockQuestion(ctx, room, sessionID)
	}
}

var (
	statsTickMu      sync.Mutex
	statsTickPending = make(map[uuid.UUID]bool)
)

func (h *Handler) sendStatsTick(ctx context.Context, room *SessionRoom, sessionID, questionID uuid.UUID) {
	statsTickMu.Lock()
	if statsTickPending[sessionID] {
		statsTickMu.Unlock()
		return
	}
	statsTickPending[sessionID] = true
	statsTickMu.Unlock()

	go func() {
		time.Sleep(time.Second)
		statsTickMu.Lock()
		delete(statsTickPending, sessionID)
		statsTickMu.Unlock()

		answered, _ := h.svc.GetAnsweredCount(ctx, sessionID, questionID)
		connected := room.connectedCount()
		room.sendTeacher(encodeMsg("question.stats_tick", evtQuestionStatsTick{
			AnsweredCount:  answered,
			ConnectedCount: connected,
		}))
	}()
}

func (h *Handler) lockQuestion(ctx context.Context, room *SessionRoom, sessionID uuid.UUID) {
	room.cancelTimer()

	_ = h.svc.SetPhase(ctx, sessionID, domain.LivePhaseQuestionLocked)

	room.mu.RLock()
	idx := room.questionIdx
	questions := room.questions
	room.mu.RUnlock()

	if idx >= len(questions) {
		return
	}

	q := questions[idx]
	answered, _ := h.svc.GetAnsweredCount(ctx, sessionID, q.ID)
	connected := room.connectedCount()

	allAnswers, _ := h.svc.GetAllAnswers(ctx, sessionID, q.ID)

	var distribution []evtOptionStat
	if q.Type == domain.QuestionTypeSingleChoice || q.Type == domain.QuestionTypeMultipleChoice {
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

	correctN, incorrectN := 0, 0
	for _, ans := range allAnswers {
		if ans.IsCorrect {
			correctN++
		} else {
			incorrectN++
		}
	}

	stats := evtQuestionStatsTick{
		AnsweredCount:  answered,
		ConnectedCount: connected,
		CorrectCount:   correctN,
		IncorrectCount: incorrectN,
	}

	room.sendTeacher(encodeMsg("question.locked", evtQuestionLocked{
		QuestionID:   q.ID.String(),
		Stats:        stats,
		Distribution: distribution,
	}))

	room.mu.RLock()
	for attemptID, c := range room.students {
		ans := allAnswers[attemptID]
		c.send(encodeMsg("question.locked", evtQuestionLocked{
			QuestionID:   q.ID.String(),
			Stats:        stats,
			Distribution: distribution,
			MyResult: &evtMyResult{
				IsCorrect: ans.IsCorrect,
				Score:     ans.Score,
			},
		}))
	}
	room.mu.RUnlock()
}
