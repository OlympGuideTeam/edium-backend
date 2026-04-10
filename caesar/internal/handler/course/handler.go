package course

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
	service courseService
}

func NewHandler(service courseService) *Handler {
	return &Handler{service: service}
}

func userIDFromCtx(c *gin.Context) (uuid.UUID, bool) {
	id, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		httpx.WriteError(c, apperr.ErrUnauthorized)
	}
	return id, ok
}

func parseCourseID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("courseId"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) CreateCourse(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	var req dto.CreateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	classID, err := uuid.Parse(req.ClassID)
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	id, err := h.service.CreateCourse(c.Request.Context(), classID, userID, req.Title)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CreateCourseResponse{ID: id.String()})
}

func (h *Handler) GetCourse(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	courseID, ok := parseCourseID(c)
	if !ok {
		return
	}

	detail, err := h.service.GetCourse(c.Request.Context(), courseID, userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	resp := dto.CourseDetailResponse{
		ID:           detail.ID.String(),
		Title:        detail.Title,
		TeacherName:  detail.TeacherName,
		ModuleCount:  detail.ModuleCount,
		ElementCount: detail.ElementCount,
		IsTeacher:    detail.IsTeacher,
		Modules:      make([]dto.ModuleItem, 0, len(detail.Modules)),
	}
	for _, m := range detail.Modules {
		items := make([]dto.CourseItemDTO, 0, len(m.Items))
		for _, it := range m.Items {
			item := dto.CourseItemDTO{
				ID:         it.ID.String(),
				RefID:      it.RefID.String(),
				Type:       string(it.Type),
				OrderIndex: it.OrderIndex,
			}
			if it.AttemptID != nil {
				s := it.AttemptID.String()
				item.AttemptID = &s
			}
			item.Score = it.Score
			items = append(items, item)
		}
		resp.Modules = append(resp.Modules, dto.ModuleItem{
			ID:           m.ID.String(),
			Title:        m.Title,
			ElementCount: m.ElementCount,
			Items:        items,
		})
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetClassCourses(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	classID, err := uuid.Parse(c.Param("classId"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	items, err := h.service.GetClassCourses(c.Request.Context(), classID, userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	resp := dto.CourseListResponse{Courses: make([]dto.CourseSummary, 0, len(items))}
	for _, item := range items {
		resp.Courses = append(resp.Courses, dto.CourseSummary{
			ID:           item.ID.String(),
			Title:        item.Title,
			TeacherName:  item.TeacherName,
			ModuleCount:  item.ModuleCount,
			ElementCount: item.ElementCount,
			IsTeacher:    item.IsTeacher,
		})
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UpdateCourse(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	courseID, ok := parseCourseID(c)
	if !ok {
		return
	}

	var req dto.UpdateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	if err := h.service.UpdateCourse(c.Request.Context(), courseID, userID, req.Title); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteCourse(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	courseID, ok := parseCourseID(c)
	if !ok {
		return
	}

	if err := h.service.DeleteCourse(c.Request.Context(), courseID, userID); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
