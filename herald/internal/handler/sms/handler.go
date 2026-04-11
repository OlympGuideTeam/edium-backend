package smshandler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler обслуживает HTTP-эндпоинты для Android SMS-шлюза.
// GET  /herald/v1/sms/tasks        — поллинг ожидающих задач
// POST /herald/v1/sms/tasks/:id/ack — подтверждение отправки
type Handler struct {
	tasks  SMSTaskRepository
	apiKey string
}

func NewHandler(tasks SMSTaskRepository, apiKey string) *Handler {
	return &Handler{tasks: tasks, apiKey: apiKey}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	sms := rg.Group("/sms", h.authMiddleware())
	sms.GET("/tasks", h.listTasks)
	sms.POST("/tasks/:id/ack", h.ackTask)
}

func (h *Handler) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if token == "" || token != h.apiKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

type smsTaskResponse struct {
	ID    string `json:"id"`
	Phone string `json:"phone"`
	Text  string `json:"text"`
}

func (h *Handler) listTasks(c *gin.Context) {
	tasks, err := h.tasks.ListPending(c.Request.Context(), 10)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "sms-handler: ListPending", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	resp := make([]smsTaskResponse, 0, len(tasks))
	for _, t := range tasks {
		resp = append(resp, smsTaskResponse{
			ID:    t.ID.String(),
			Phone: t.Phone,
			Text:  t.Text,
		})
	}
	c.JSON(http.StatusOK, resp)
}

type ackRequest struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (h *Handler) ackTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req ackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	if err := h.tasks.Ack(c.Request.Context(), id, req.Success, req.Error); err != nil {
		slog.ErrorContext(c.Request.Context(), "sms-handler: Ack", "task_id", id, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.Status(http.StatusNoContent)
}
