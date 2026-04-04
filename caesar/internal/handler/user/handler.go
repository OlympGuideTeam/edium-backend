package user

import (
	"net/http"

	"caesar/internal/middleware"
	"caesar/internal/pkg/apperr"
	"caesar/internal/pkg/httpx"
	"caesar/internal/transport/dto"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service userService
}

func NewHandler(service userService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetMe(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		httpx.WriteError(c, apperr.ErrUnauthorized)
		return
	}

	u, err := h.service.GetMe(c.Request.Context(), userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.UserProfileResponse{
		ID:      u.ID.String(),
		Name:    u.Name,
		Surname: u.Surname,
	})
}

func (h *Handler) UpdateMe(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		httpx.WriteError(c, apperr.ErrUnauthorized)
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	if req.Name == nil && req.Surname == nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	if err := h.service.UpdateMe(c.Request.Context(), userID, req.Name, req.Surname); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteMe(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		httpx.WriteError(c, apperr.ErrUnauthorized)
		return
	}

	if err := h.service.DeleteMe(c.Request.Context(), userID); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
