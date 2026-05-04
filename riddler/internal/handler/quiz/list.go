package quiz

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"riddler/internal/domain"
	"riddler/internal/pkg/apperr"
	"riddler/internal/pkg/httpx"
	"riddler/internal/transport/dto"
)

func (h *Handler) ListQuizzes(c *gin.Context) {
	role := domain.Role(c.Query("role"))
	if role != domain.RoleTeacher && role != domain.RoleStudent {
		httpx.WriteError(c, apperr.ErrInvalidRole)
		return
	}

	search := c.Query("search")

	items, err := h.service.ListQuizzes(c.Request.Context(), role, &search)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	result := make([]dto.QuizListItemResponse, len(items))
	for i := range items {
		result[i] = toListItemResponse(items[i])
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) ListMyQuizzes(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	items, err := h.service.ListMyQuizzes(c.Request.Context(), userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	result := make([]dto.QuizListItemResponse, len(items))
	for i := range items {
		result[i] = toListItemResponse(items[i])
	}

	c.JSON(http.StatusOK, result)
}
