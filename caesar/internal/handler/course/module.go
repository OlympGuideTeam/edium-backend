package course

import (
	"net/http"

	"caesar/internal/pkg/apperr"
	"caesar/internal/pkg/httpx"
	"caesar/internal/transport/dto"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) CreateModule(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	courseID, ok := parseCourseID(c)
	if !ok {
		return
	}

	var req dto.CreateModuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	id, err := h.service.CreateModule(c.Request.Context(), courseID, userID, req.Title)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CreateModuleResponse{ID: id.String()})
}

func (h *Handler) UpdateModule(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	moduleID, err := uuid.Parse(c.Param("moduleId"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	var req dto.UpdateModuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	if err := h.service.UpdateModule(c.Request.Context(), moduleID, userID, req.Title); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteModule(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	moduleID, err := uuid.Parse(c.Param("moduleId"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	if err := h.service.DeleteModule(c.Request.Context(), moduleID, userID); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
