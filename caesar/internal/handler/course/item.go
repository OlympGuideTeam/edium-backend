package course

import (
	"net/http"

	"caesar/internal/domain"
	"caesar/internal/pkg/apperr"
	"caesar/internal/pkg/httpx"
	"caesar/internal/transport/dto"

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

func (h *Handler) CreateCourseItem(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	moduleID, err := uuid.Parse(c.Param("moduleId"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	var req dto.CreateCourseItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	refID, err := uuid.Parse(req.RefID)
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	id, err := h.service.CreateCourseItem(c.Request.Context(), moduleID, userID, refID, domain.CourseItemType(req.Type), req.OrderIndex)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CreateCourseItemResponse{ID: id.String()})
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
