package user

import (
	"net/http"

	"caesar/internal/middleware"
	"caesar/internal/pkg/apperr"
	"caesar/internal/pkg/httpx"
	"caesar/internal/transport/dto"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

func (h *Handler) GetMeStatistic(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		httpx.WriteError(c, apperr.ErrUnauthorized)
		return
	}

	st, err := h.service.GetMeStatistic(c.Request.Context(), userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.UserStatisticResponse{
		ClassTeacherCount:  st.ClassTeacherCount,
		StudentCount:       st.StudentCount,
		CourseTeacherCount: st.CourseTeacherCount,
		CourseStudentCount: st.CourseStudentCount,
	})
}

func (h *Handler) GetUsersRoster(c *gin.Context) {
	var req dto.UsersRosterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	ids := make([]uuid.UUID, 0, len(req.UserIDs))
	for _, raw := range req.UserIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(c, apperr.ErrBadID)
			return
		}
		ids = append(ids, id)
	}

	users, err := h.service.GetUsersRoster(c.Request.Context(), ids)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	resp := dto.UsersRosterResponse{Users: make([]dto.UserProfileResponse, 0, len(users))}
	for _, u := range users {
		resp.Users = append(resp.Users, dto.UserProfileResponse{
			ID:      u.ID.String(),
			Name:    u.Name,
			Surname: u.Surname,
		})
	}
	c.JSON(http.StatusOK, resp)
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
