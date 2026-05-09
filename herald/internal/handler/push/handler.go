package pushhandler

import (
	"herald/internal/middleware"
	"herald/internal/pkg/apperr"
	"herald/internal/pkg/httpx"
	"herald/internal/transport/dto"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service PushService
}

func NewHandler(service PushService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.POST("/devices", h.registerDevice)
	rg.DELETE("/devices/:token", h.deleteDevice)
	rg.GET("/notifications", h.listNotifications)
	rg.GET("/notifications/count", h.countUnread)
	rg.PATCH("/notifications/:id/read", h.markRead)
}

func (h *Handler) registerDevice(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		httpx.WriteError(c, apperr.ErrUnauthorized)
		return
	}

	var req dto.RegisterDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.BadRequest(err))
		return
	}

	if err := h.service.RegisterDevice(c.Request.Context(), userID, req.FCMToken, req.Platform); err != nil {
		slog.ErrorContext(c.Request.Context(), "push-handler: RegisterDevice", "err", err)
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) deleteDevice(c *gin.Context) {
	_, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		httpx.WriteError(c, apperr.ErrUnauthorized)
		return
	}

	token := c.Param("token")
	if token == "" {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	if err := h.service.DeleteDevice(c.Request.Context(), token); err != nil {
		slog.ErrorContext(c.Request.Context(), "push-handler: DeleteDevice", "err", err)
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) listNotifications(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		httpx.WriteError(c, apperr.ErrUnauthorized)
		return
	}

	notifications, err := h.service.ListNotifications(c.Request.Context(), userID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "push-handler: ListNotifications", "err", err)
		httpx.WriteError(c, err)
		return
	}

	resp := make([]dto.NotificationResponse, 0, len(notifications))
	for _, n := range notifications {
		item := dto.NotificationResponse{
			ID:        n.ID.String(),
			Title:     n.Title,
			Body:      n.Body,
			CreatedAt: n.CreatedAt,
			IsRead:    n.IsRead,
		}
		if n.Data != nil {
			item.Data = &dto.NotificationDataResponse{
				Route: n.Data.Route,
				Role:  n.Data.Role,
			}
		}
		resp = append(resp, item)
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) countUnread(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		httpx.WriteError(c, apperr.ErrUnauthorized)
		return
	}

	count, err := h.service.CountUnread(c.Request.Context(), userID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "push-handler: CountUnread", "err", err)
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

func (h *Handler) markRead(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		httpx.WriteError(c, apperr.ErrUnauthorized)
		return
	}

	notifID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	if err := h.service.MarkRead(c.Request.Context(), userID, notifID); err != nil {
		slog.ErrorContext(c.Request.Context(), "push-handler: MarkRead", "err", err)
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
