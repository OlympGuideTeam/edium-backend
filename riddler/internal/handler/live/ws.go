package live

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"riddler/internal/domain"
	"riddler/internal/pkg/apperr"
	"riddler/internal/pkg/httpx"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *Handler) WebSocket(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	token := c.Query("token")
	if token == "" {
		httpx.WriteError(c, apperr.ErrWsTokenInvalid)
		return
	}

	payload, err := h.svc.ConsumeWsToken(c.Request.Context(), sessionID, token)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	if payload == nil {
		httpx.WriteError(c, apperr.ErrWsTokenInvalid)
		return
	}

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	var userID, attemptID uuid.UUID
	if payload.UserID != "" {
		userID, _ = uuid.Parse(payload.UserID)
	}
	if payload.AttemptID != "" {
		attemptID, _ = uuid.Parse(payload.AttemptID)
	}

	conn := newConn(ws, sessionID, payload.Role, userID, attemptID)
	room := h.hub.getOrCreate(sessionID)

	if payload.Role == domain.RoleTeacher {
		h.connectTeacher(c.Request.Context(), conn, room, sessionID)
	} else {
		kicked, err := h.svc.IsKicked(c.Request.Context(), sessionID, attemptID)
		if err != nil || kicked {
			_ = ws.Close()
			return
		}
		h.connectStudent(c.Request.Context(), conn, room, sessionID)
	}
}

func (h *Handler) connectTeacher(ctx context.Context, conn *Conn, room *SessionRoom, sessionID uuid.UUID) {
	room.setTeacher(conn)
	go conn.writePump()

	if len(room.questions) == 0 {
		questions, err := h.svc.GetQuestions(ctx, sessionID)
		if err == nil {
			room.mu.Lock()
			room.questions = questions
			room.mu.Unlock()
		}
	}

	snapshot := h.buildSnapshot(ctx, conn, room, sessionID)
	conn.send(snapshot)

	h.readLoop(ctx, conn, room, sessionID)

	room.removeTeacher(conn)
	conn.closeWS()
}

func (h *Handler) connectStudent(ctx context.Context, conn *Conn, room *SessionRoom, sessionID uuid.UUID) {
	room.addStudent(conn.attemptID, conn)
	go conn.writePump()

	snapshot := h.buildSnapshot(ctx, conn, room, sessionID)
	conn.send(snapshot)

	participants, _ := h.svc.GetParticipants(ctx, sessionID)
	joined := encodeMsg("lobby.participant_joined", evtLobbyParticipant{
		AttemptID: conn.attemptID.String(),
	})
	for _, p := range participants {
		if p.AttemptID == conn.attemptID {
			s := ""
			if p.Name != nil {
				s = *p.Name
			}
			uid := ""
			if p.UserID != nil {
				uid = p.UserID.String()
			}
			joined = encodeMsg("lobby.participant_joined", evtLobbyParticipant{
				AttemptID: conn.attemptID.String(),
				UserID:    strPtr(uid),
				Name:      strPtr(s),
			})
			break
		}
	}
	room.broadcastAll(joined)

	h.readLoop(ctx, conn, room, sessionID)

	room.removeStudent(conn.attemptID)
	conn.closeWS()

	go h.gracePeriod(conn.attemptID, room, sessionID)
}

func (h *Handler) readLoop(ctx context.Context, conn *Conn, room *SessionRoom, sessionID uuid.UUID) {
	for {
		_, raw, err := conn.ws.ReadMessage()
		if err != nil {
			return
		}
		var msg wsMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		if conn.role == domain.RoleTeacher {
			h.handleTeacherCmd(ctx, conn, room, sessionID, msg)
		} else {
			h.handleStudentCmd(ctx, conn, room, sessionID, msg)
		}
	}
}

func (h *Handler) gracePeriod(attemptID uuid.UUID, room *SessionRoom, sessionID uuid.UUID) {
	time.Sleep(2 * time.Second)
	if room.getStudent(attemptID) != nil {
		return
	}
	ctx := context.Background()
	_ = h.svc.SetParticipantStatus(ctx, sessionID, attemptID, "disconnected")
	room.broadcastAll(encodeMsg("lobby.participant_left", evtParticipantKicked{
		AttemptID: attemptID.String(),
	}))
}

func (h *Handler) buildSnapshot(ctx context.Context, conn *Conn, room *SessionRoom, sessionID uuid.UUID) []byte {
	meta, _ := h.svc.GetSessionMeta(ctx, sessionID)
	phase := domain.LivePhase("")
	if meta != nil {
		phase = meta.Phase
	}

	questionTotal := 0
	if meta != nil {
		questionTotal = meta.QuestionCount
	}
	snap := evtStateSnapshot{Phase: string(phase), QuestionTotal: questionTotal}

	switch phase {
	case domain.LivePhaseLobby:
		participants, _ := h.svc.GetParticipants(ctx, sessionID)
		for _, p := range participants {
			row := evtLobbyParticipant{AttemptID: p.AttemptID.String()}
			if p.UserID != nil {
				s := p.UserID.String()
				row.UserID = &s
			}
			row.Name = p.Name
			snap.Participants = append(snap.Participants, row)
		}

	case domain.LivePhaseQuestionActive, domain.LivePhaseQuestionLocked:
		idx, deadline, err := h.svc.GetCurrentQuestion(ctx, sessionID)
		if err != nil {
			break
		}
		room.mu.RLock()
		questions := room.questions
		room.mu.RUnlock()

		if idx < len(questions) {
			q := questions[idx]
			snap.QuestionIdx = idx
			snap.TimeLimitSec = meta.QuestionTimeLimitSec
			if !deadline.IsZero() {
				snap.DeadlineAt = deadline.UTC().Format(time.RFC3339)
			}
			eq := buildQuestion(q, conn.role == domain.RoleTeacher)
			snap.CurrentQuestion = &eq
		}
	}

	return encodeMsg("state.snapshot", snap)
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
