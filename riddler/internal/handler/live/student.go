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
		AttemptID:   conn.attemptID.String(),
		QuestionID:  questionID.String(),
		IsCorrect:   isCorrect,
		TimeTakenMs: timeTakenMs,
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

		room.mu.RLock()
		idx := room.questionIdx
		questions := room.questions
		room.mu.RUnlock()

		if idx >= len(questions) || questions[idx].ID != questionID {
			return
		}

		allAnswers, _ := h.svc.GetAllAnswers(ctx, sessionID, questionID)
		connected := room.connectedCount()
		stats, distribution := buildStats(questions[idx], allAnswers, connected)
		stats.Distribution = distribution
		room.sendTeacher(encodeMsg("question.stats_tick", stats))
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
	connected := room.connectedCount()

	allAnswers, _ := h.svc.GetAllAnswers(ctx, sessionID, q.ID)
	stats, distribution := buildStats(q, allAnswers, connected)
	correctAnswer := buildCorrectAnswer(q)
	wordCloud := buildWordCloud(q, allAnswers)

	room.sendTeacher(encodeMsg("question.locked", evtQuestionLocked{
		QuestionID:    q.ID.String(),
		Stats:         stats,
		Distribution:  distribution,
		CorrectAnswer: correctAnswer,
		WordCloud:     wordCloud,
	}))

	room.mu.RLock()
	for attemptID, c := range room.students {
		ans := allAnswers[attemptID]
		c.send(encodeMsg("question.locked", evtQuestionLocked{
			QuestionID:    q.ID.String(),
			Stats:         stats,
			Distribution:  distribution,
			CorrectAnswer: correctAnswer,
			WordCloud:     wordCloud,
			MyResult: &evtMyResult{
				IsCorrect: ans.IsCorrect,
				Score:     ans.Score,
			},
		}))
	}
	room.mu.RUnlock()
}
