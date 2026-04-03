package user

import (
	"caesar/internal/middleware"
	"caesar/internal/pkg/apperr"
	"caesar/internal/pkg/httpx"
	"caesar/internal/transport/dto"
	"net/http"

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
		httpx.WriteError(c, apperr.New("UNAUTHORIZED", "Требуется авторизация", 401))
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
