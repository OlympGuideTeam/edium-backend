package course

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

	drafts := make([]dto.CourseDraftDTO, 0, len(detail.Drafts))
	for _, d := range detail.Drafts {
		drafts = append(drafts, dto.CourseDraftDTO{
			ID:             d.ID.String(),
			QuizTemplateID: d.QuizTemplateID.String(),
			Type:           string(domain.CourseItemTypeQuiz),
			Payload:        d.Payload,
		})
	}

	resp := dto.CourseDetailResponse{
		ID:           detail.ID.String(),
		Title:        detail.Title,
		TeacherName:  detail.TeacherName,
		ModuleCount:  detail.ModuleCount,
		ElementCount: detail.ElementCount,
		IsTeacher:    detail.IsTeacher,
		Drafts:       drafts,
		Modules:      make([]dto.ModuleItem, 0, len(detail.Modules)),
	}
	for _, m := range detail.Modules {
		resp.Modules = append(resp.Modules, dto.ModuleItem{
			ID:           m.ID.String(),
			Title:        m.Title,
			ElementCount: m.ElementCount,
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

func (h *Handler) GetCourseSheet(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	courseID, ok := parseCourseID(c)
	if !ok {
		return
	}

	sheet, err := h.service.GetCourseSheet(c.Request.Context(), courseID, userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	items := make([]dto.SheetColumn, len(sheet.Items))
	for i, item := range sheet.Items {
		items[i] = dto.SheetColumn{
			ID:    item.ID.String(),
			RefID: item.RefID.String(),
		}
	}

	rows := make([]dto.SheetRow, len(sheet.Students))
	for i, row := range sheet.Students {
		scores := make([]dto.SheetScore, len(row.Scores))
		for j, sc := range row.Scores {
			scores[j] = dto.SheetScore{
				ItemID: sc.ItemID.String(),
				Score:  sc.Score,
			}
		}
		rows[i] = dto.SheetRow{
			StudentID:   row.StudentID.String(),
			StudentName: row.StudentName,
			Scores:      scores,
		}
	}

	c.JSON(http.StatusOK, dto.CourseSheet{
		Items:    items,
		Students: rows,
	})
}
