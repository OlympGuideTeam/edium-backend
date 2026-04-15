package course

import (
	"net/http"

	"caesar/internal/pkg/apperr"
	"caesar/internal/pkg/httpx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func parseItemID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) DeleteCourseItem(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	itemID, ok := parseItemID(c)
	if !ok {
		return
	}

	if err := h.service.DeleteCourseItem(c.Request.Context(), itemID, userID); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
