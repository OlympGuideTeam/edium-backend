package class

import (
	"net/http"

	"caesar/internal/pkg/apperr"
	"caesar/internal/pkg/httpx"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) RemoveMember(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	classID, ok := parseClassID(c)
	if !ok {
		return
	}

	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	if err := h.service.RemoveMember(c.Request.Context(), classID, userID, targetUserID); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
