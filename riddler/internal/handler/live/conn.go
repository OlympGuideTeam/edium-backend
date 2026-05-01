package live

import (
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"riddler/internal/domain"
)

const sendBufSize = 64

type Conn struct {
	ws        *websocket.Conn
	sendCh    chan []byte
	done      chan struct{}
	closeOnce sync.Once
	sessionID uuid.UUID
	role      domain.Role
	userID    uuid.UUID // teacher
	attemptID uuid.UUID // student
}

func newConn(ws *websocket.Conn, sessionID uuid.UUID, role domain.Role, userID, attemptID uuid.UUID) *Conn {
	return &Conn{
		ws:        ws,
		sendCh:    make(chan []byte, sendBufSize),
		done:      make(chan struct{}),
		sessionID: sessionID,
		role:      role,
		userID:    userID,
		attemptID: attemptID,
	}
}

// send кладёт сообщение в буфер; если соединение закрыто или буфер полон — дропает.
func (c *Conn) send(msg []byte) {
	select {
	case <-c.done:
	case c.sendCh <- msg:
	default:
	}
}

// closeWS завершает соединение ровно один раз: закрывает done и WS.
func (c *Conn) closeWS() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.ws.Close()
	})
}

// writePump пишет сообщения из буфера в WS; завершается при закрытии done или ошибке записи.
func (c *Conn) writePump() {
	defer c.closeWS()
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
