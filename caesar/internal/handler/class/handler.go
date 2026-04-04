package class

import (
	"net/http"

	"caesar/internal/domain"
	"caesar/internal/middleware"
	"caesar/internal/pkg/apperr"
	"caesar/internal/pkg/httpx"
	"caesar/internal/transport/dto"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service classService
}

func NewHandler(service classService) *Handler {
	return &Handler{service: service}
}

func userIDFromCtx(c *gin.Context) (uuid.UUID, bool) {
	id, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		httpx.WriteError(c, apperr.ErrUnauthorized)
	}
	return id, ok
}

func parseClassID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("classId"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) GetMyClasses(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	roleParam := c.Query("role")
	var role domain.ClassMemberRole
	switch roleParam {
	case string(domain.ClassMemberRoleTeacher):
		role = domain.ClassMemberRoleTeacher
	case string(domain.ClassMemberRoleStudent):
		role = domain.ClassMemberRoleStudent
	default:
		httpx.WriteError(c, apperr.ErrInvalidRole)
		return
	}

	classes, err := h.service.GetMyClasses(c.Request.Context(), userID, role)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	resp := dto.ClassListResponse{Classes: make([]dto.ClassSummary, 0, len(classes))}
	for _, cl := range classes {
		resp.Classes = append(resp.Classes, dto.ClassSummary{
			ID:           cl.ID.String(),
			Title:        cl.Title,
			OwnerName:    cl.OwnerName,
			StudentCount: cl.StudentCount,
			IsOwner:      cl.IsOwner,
		})
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) CreateClass(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	var req dto.CreateClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	id, err := h.service.CreateClass(c.Request.Context(), userID, req.Title)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CreateClassResponse{ID: id.String()})
}

func (h *Handler) GetClass(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	classID, ok := parseClassID(c)
	if !ok {
		return
	}

	detail, err := h.service.GetClass(c.Request.Context(), classID, userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	resp := dto.ClassDetailResponse{
		ID:        detail.ID.String(),
		Title:     detail.Title,
		OwnerName: detail.OwnerName,
		IsOwner:   detail.IsOwner,
		Teachers:  make([]dto.MemberShort, 0, len(detail.Teachers)),
		Students:  make([]dto.MemberShort, 0, len(detail.Students)),
	}
	for _, m := range detail.Teachers {
		resp.Teachers = append(resp.Teachers, dto.MemberShort{
			ID:      m.UserID.String(),
			Name:    m.Name,
			Surname: m.Surname,
		})
	}
	for _, m := range detail.Students {
		resp.Students = append(resp.Students, dto.MemberShort{
			ID:      m.UserID.String(),
			Name:    m.Name,
			Surname: m.Surname,
		})
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UpdateClass(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	classID, ok := parseClassID(c)
	if !ok {
		return
	}

	var req dto.UpdateClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	if err := h.service.UpdateClass(c.Request.Context(), classID, userID, req.Title); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteClass(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	classID, ok := parseClassID(c)
	if !ok {
		return
	}

	if err := h.service.DeleteClass(c.Request.Context(), classID, userID); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
