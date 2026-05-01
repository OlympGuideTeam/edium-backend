package live

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

type Hub struct {
	mu       sync.RWMutex
	sessions map[uuid.UUID]*SessionRoom
}

func newHub() *Hub {
	return &Hub{sessions: make(map[uuid.UUID]*SessionRoom)}
}

func (h *Hub) getOrCreate(sessionID uuid.UUID) *SessionRoom {
	h.mu.Lock()
	defer h.mu.Unlock()
	if r, ok := h.sessions[sessionID]; ok {
		return r
	}
	r := &SessionRoom{students: make(map[uuid.UUID]*Conn)}
	h.sessions[sessionID] = r
	return r
}

func (h *Hub) remove(sessionID uuid.UUID) {
	h.mu.Lock()
	delete(h.sessions, sessionID)
	h.mu.Unlock()
}

type SessionRoom struct {
	mu                sync.RWMutex
	teacher           *Conn
	students          map[uuid.UUID]*Conn // attempt_id → conn
	timerCancel       func()
	questions         []domain.QuestionWithOptions
	questionIdx       int
	questionStartedAt time.Time
}

func (r *SessionRoom) setTeacher(c *Conn) {
	r.mu.Lock()
	r.teacher = c
	r.mu.Unlock()
}

func (r *SessionRoom) removeTeacher(c *Conn) {
	r.mu.Lock()
	if r.teacher == c {
		r.teacher = nil
	}
	r.mu.Unlock()
}

func (r *SessionRoom) addStudent(attemptID uuid.UUID, c *Conn) {
	r.mu.Lock()
	r.students[attemptID] = c
	r.mu.Unlock()
}

func (r *SessionRoom) removeStudent(attemptID uuid.UUID) {
	r.mu.Lock()
	delete(r.students, attemptID)
	r.mu.Unlock()
}

func (r *SessionRoom) getStudent(attemptID uuid.UUID) *Conn {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.students[attemptID]
}

func (r *SessionRoom) connectedCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.students)
}

func (r *SessionRoom) broadcastAll(msg []byte) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.teacher != nil {
		r.teacher.send(msg)
	}
	for _, c := range r.students {
		c.send(msg)
	}
}

func (r *SessionRoom) broadcastStudents(msg []byte) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.students {
		c.send(msg)
	}
}

func (r *SessionRoom) sendTeacher(msg []byte) {
	r.mu.RLock()
	t := r.teacher
	r.mu.RUnlock()
	if t != nil {
		t.send(msg)
	}
}

func (r *SessionRoom) cancelTimer() {
	r.mu.Lock()
	if r.timerCancel != nil {
		r.timerCancel()
		r.timerCancel = nil
	}
	r.mu.Unlock()
}

func (r *SessionRoom) setTimer(cancel func()) {
	r.mu.Lock()
	r.timerCancel = cancel
	r.mu.Unlock()
}

func (r *SessionRoom) currentQuestion() domain.QuestionWithOptions {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.questions[r.questionIdx]
}
