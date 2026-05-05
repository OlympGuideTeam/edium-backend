package live

import (
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"riddler/internal/middleware"
	"riddler/internal/pkg/apperr"
	"riddler/internal/pkg/httpx"
)

func (h *Handler) StudentWebSocket(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		httpx.WriteError(c, apperr.ErrUnauthorized)
		return
	}

	rawIDs := c.Query("course_ids")
	if rawIDs == "" {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	parts := strings.Split(rawIDs, ",")
	courseIDs := make([]uuid.UUID, 0, len(parts))
	for _, p := range parts {
		id, err := uuid.Parse(strings.TrimSpace(p))
		if err != nil {
			httpx.WriteError(c, apperr.ErrBadID)
			return
		}
		courseIDs = append(courseIDs, id)
	}

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	conn := newStudentConn(ws)
	go conn.writePump()

	snapshot, err := h.svc.GetActiveCourseLiveSessions(c.Request.Context(), courseIDs)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "student ws: snapshot error", "user_id", userID, "err", err)
	}

	snapshotItems := make([]map[string]any, 0, len(snapshot))
	for _, s := range snapshot {
		snapshotItems = append(snapshotItems, map[string]any{
			"session_id":              s.SessionID.String(),
			"course_id":               s.CourseID.String(),
			"quiz_title":              s.QuizTitle,
			"question_time_limit_sec": s.QuestionTimeLimitSec,
		})
	}
	snapshotMsg, _ := json.Marshal(map[string]any{
		"type": "snapshot",
		"data": snapshotItems,
	})
	conn.send(snapshotMsg)

	h.studentHub.register(conn, courseIDs)
	defer func() {
		h.studentHub.unregister(conn, courseIDs)
		conn.close()
	}()

	ws.SetReadLimit(512)
	for {
		if _, _, err := ws.ReadMessage(); err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			}
			return
		}
	}
}
