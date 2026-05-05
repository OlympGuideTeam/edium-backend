package live

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type StudentHub struct {
	mu             sync.RWMutex
	courseConns    map[uuid.UUID]map[*studentConn]struct{}
	sessionCourses map[uuid.UUID]uuid.UUID
}

func NewStudentHub() *StudentHub {
	return &StudentHub{
		courseConns:    make(map[uuid.UUID]map[*studentConn]struct{}),
		sessionCourses: make(map[uuid.UUID]uuid.UUID),
	}
}

type studentConn struct {
	ws        *websocket.Conn
	sendCh    chan []byte
	done      chan struct{}
	closeOnce sync.Once
}

func newStudentConn(ws *websocket.Conn) *studentConn {
	return &studentConn{
		ws:     ws,
		sendCh: make(chan []byte, 32),
		done:   make(chan struct{}),
	}
}

func (c *studentConn) send(msg []byte) {
	select {
	case <-c.done:
	case c.sendCh <- msg:
	default:
	}
}

func (c *studentConn) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.ws.Close()
	})
}

func (c *studentConn) writePump() {
	defer c.close()
	for {
		select {
		case <-c.done:
			return
		case msg := <-c.sendCh:
			if err := c.ws.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}
}

func (h *StudentHub) register(conn *studentConn, courseIDs []uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, id := range courseIDs {
		if h.courseConns[id] == nil {
			h.courseConns[id] = make(map[*studentConn]struct{})
		}
		h.courseConns[id][conn] = struct{}{}
	}
}

func (h *StudentHub) unregister(conn *studentConn, courseIDs []uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, id := range courseIDs {
		delete(h.courseConns[id], conn)
		if len(h.courseConns[id]) == 0 {
			delete(h.courseConns, id)
		}
	}
}

func (h *StudentHub) NotifyLobbyOpened(sessionID, courseID uuid.UUID, quizTitle string, questionTimeLimitSec int) {
	msg, _ := json.Marshal(wsMessage{
		Type: "lobby_opened",
		Data: json.RawMessage(mustMarshal(map[string]any{
			"session_id":             sessionID.String(),
			"course_id":              courseID.String(),
			"quiz_title":             quizTitle,
			"question_time_limit_sec": questionTimeLimitSec,
		})),
	})

	h.mu.Lock()
	h.sessionCourses[sessionID] = courseID
	conns := h.courseConns[courseID]
	h.mu.Unlock()

	for c := range conns {
		c.send(msg)
	}
}

func (h *StudentHub) NotifyLobbyClosed(sessionID uuid.UUID) {
	msg, _ := json.Marshal(wsMessage{
		Type: "lobby_closed",
		Data: json.RawMessage(mustMarshal(map[string]any{
			"session_id": sessionID.String(),
		})),
	})

	h.mu.Lock()
	courseID, ok := h.sessionCourses[sessionID]
	if ok {
		delete(h.sessionCourses, sessionID)
	}
	conns := h.courseConns[courseID]
	h.mu.Unlock()

	if !ok {
		return
	}
	for c := range conns {
		c.send(msg)
	}
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
