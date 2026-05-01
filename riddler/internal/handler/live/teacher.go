package live

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

func (h *Handler) handleTeacherCmd(ctx context.Context, conn *Conn, room *SessionRoom, sessionID uuid.UUID, msg wsMessage) {
	switch msg.Type {
	case "teacher.start_quiz":
		h.handleStartQuiz(ctx, conn, room, sessionID)
	case "teacher.next_question":
		h.handleNextQuestion(ctx, conn, room, sessionID)
	case "teacher.kick_participant":
		var cmd cmdKickParticipant
		if err := json.Unmarshal(msg.Data, &cmd); err != nil {
			return
		}
		attemptID, err := uuid.Parse(cmd.AttemptID)
		if err != nil {
			return
		}
		h.handleKick(ctx, room, sessionID, attemptID)
	}
}

func (h *Handler) handleStartQuiz(ctx context.Context, _ *Conn, room *SessionRoom, sessionID uuid.UUID) {
	room.mu.Lock()
	if len(room.questions) == 0 {
		room.mu.Unlock()
		return
	}
	room.questionIdx = 0
	room.mu.Unlock()

	h.startQuestion(ctx, room, sessionID)
}

func (h *Handler) handleNextQuestion(ctx context.Context, _ *Conn, room *SessionRoom, sessionID uuid.UUID) {
	room.cancelTimer()

	room.mu.Lock()
	nextIdx := room.questionIdx + 1
	total := len(room.questions)
	room.mu.Unlock()

	if nextIdx >= total {
		h.completeQuiz(ctx, room, sessionID)
		return
	}

	room.mu.Lock()
	room.questionIdx = nextIdx
	room.mu.Unlock()

	h.startQuestion(ctx, room, sessionID)
}

func (h *Handler) startQuestion(ctx context.Context, room *SessionRoom, sessionID uuid.UUID) {
	room.mu.RLock()
	idx := room.questionIdx
	questions := room.questions
	meta, _ := h.svc.GetSessionMeta(ctx, sessionID)
	room.mu.RUnlock()

	if idx >= len(questions) {
		return
	}

	timeLimitSec := 30
	if meta != nil && meta.QuestionTimeLimitSec > 0 {
		timeLimitSec = meta.QuestionTimeLimitSec
	}

	now := time.Now()
	deadline := now.Add(time.Duration(timeLimitSec) * time.Second)

	room.mu.Lock()
	room.questionStartedAt = now
	room.mu.Unlock()

	_ = h.svc.SetPhase(ctx, sessionID, domain.LivePhaseQuestionActive)
	_ = h.svc.SetCurrentQuestion(ctx, sessionID, idx, deadline)

	q := questions[idx]

	room.broadcastAll(encodeMsg("quiz.started", nil))

	room.sendTeacher(encodeMsg("question.started", evtQuestionStarted{
		QuestionIdx:  idx,
		Question:     buildQuestion(q, true),
		TimeLimitSec: timeLimitSec,
		DeadlineAt:   deadline.UTC().Format(time.RFC3339),
	}))
	room.broadcastStudents(encodeMsg("question.started", evtQuestionStarted{
		QuestionIdx:  idx,
		Question:     buildQuestion(q, false),
		TimeLimitSec: timeLimitSec,
		DeadlineAt:   deadline.UTC().Format(time.RFC3339),
	}))

	timerCtx, cancel := context.WithCancel(ctx)
	room.setTimer(cancel)
	go func() {
		select {
		case <-timerCtx.Done():
		case <-time.After(time.Duration(timeLimitSec) * time.Second):
			h.lockQuestion(context.Background(), room, sessionID)
		}
	}()
}

func (h *Handler) handleKick(ctx context.Context, room *SessionRoom, sessionID, attemptID uuid.UUID) {
	target := room.getStudent(attemptID)

	_ = h.svc.SetParticipantStatus(ctx, sessionID, attemptID, "kicked")
	_ = h.svc.SetKicked(ctx, attemptID)

	room.broadcastAll(encodeMsg("participant.kicked", evtParticipantKicked{
		AttemptID: attemptID.String(),
	}))

	if target != nil {
		target.send(encodeMsg("you_were_kicked", nil))
		target.closeWS()
	}
}

func (h *Handler) completeQuiz(ctx context.Context, room *SessionRoom, sessionID uuid.UUID) {
	room.broadcastAll(encodeMsg("quiz.completed", nil))

	_ = h.svc.CompleteLiveSession(ctx, sessionID)

	room.mu.RLock()
	for _, c := range room.students {
		c.closeWS()
	}
	if room.teacher != nil {
		room.teacher.closeWS()
	}
	room.mu.RUnlock()

	h.hub.remove(sessionID)
}

func buildQuestion(q domain.QuestionWithOptions, withCorrect bool) evtQuestion {
	opts := make([]evtAnswerOption, len(q.Options))
	for i, o := range q.Options {
		opts[i] = evtAnswerOption{
			ID:   o.ID.String(),
			Text: o.Text,
		}
		if withCorrect {
			opts[i].IsCorrect = o.IsCorrect
		}
	}
	eq := evtQuestion{
		ID:       q.ID.String(),
		Type:     string(q.Type),
		Text:     q.Text,
		MaxScore: q.MaxScore,
		Options:  opts,
		Metadata: q.Metadata,
	}
	if q.ImageID != nil {
		s := q.ImageID.String()
		eq.ImageID = &s
	}
	return eq
}
