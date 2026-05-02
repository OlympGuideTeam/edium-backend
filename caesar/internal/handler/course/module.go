package course

import (
	"net/http"

	"caesar/internal/pkg/apperr"
	"caesar/internal/pkg/httpx"
	"caesar/internal/transport/dto"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) GetModule(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	moduleID, err := uuid.Parse(c.Param("moduleId"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	m, err := h.service.GetModule(c.Request.Context(), moduleID, userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	items := make([]dto.CourseItemDTO, 0, len(m.Items))
	for _, it := range m.Items {
		item := dto.CourseItemDTO{
			ID:       it.ID.String(),
			ObjectID: it.ObjectID.String(),
			Type:     string(it.Type),
			Payload:  it.Payload,
		}
		if it.AttemptID != nil {
			s := it.AttemptID.String()
			item.AttemptID = &s
		}
		item.Score = it.Score
		items = append(items, item)
	}

	c.JSON(http.StatusOK, dto.ModuleDetailResponse{
		ID:           m.ID.String(),
		Title:        m.Title,
		ElementCount: m.ElementCount,
		Items:        items,
	})
}

func (h *Handler) GetModuleRoster(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	moduleID, err := uuid.Parse(c.Param("moduleId"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	members, err := h.service.GetModuleRoster(c.Request.Context(), moduleID, userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	resp := dto.ModuleRosterResponse{Members: make([]dto.MemberShort, 0, len(members))}
	for _, m := range members {
		resp.Members = append(resp.Members, dto.MemberShort{
			ID:      m.UserID.String(),
			Name:    m.Name,
			Surname: m.Surname,
		})
	}
	c.JSON(http.StatusOK, resp)
}

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

func (h *Handler) ReorderModules(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	courseID, ok := parseCourseID(c)
	if !ok {
		return
	}

	var req dto.ReorderModulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	moduleIDs := make([]uuid.UUID, 0, len(req.ModuleIDs))
	for _, raw := range req.ModuleIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(c, apperr.ErrBadID)
			return
		}
		moduleIDs = append(moduleIDs, id)
	}

	if err := h.service.ReorderModules(c.Request.Context(), courseID, userID, moduleIDs); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusOK)
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
