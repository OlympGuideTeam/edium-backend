package quiz

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/pkg/apperr"
	"riddler/internal/pkg/httpx"
	"riddler/internal/transport/dto"
)

func (h *Handler) CreateQuiz(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	var req dto.CreateQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	settings, err := quizSettingsFromDTO(req.DefaultSettings)
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	var attachToCourse *uuid.UUID
	if req.AttachToCourse != nil {
		cid, err := uuid.Parse(req.AttachToCourse.CourseID)
		if err != nil {
			httpx.WriteError(c, apperr.ErrBadID)
			return
		}
		attachToCourse = &cid
	}

	id, err := h.service.CreateQuiz(c.Request.Context(), userID, req.Title, req.Description, settings, attachToCourse)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CreateQuizResponse{ID: id.String()})
}

func (h *Handler) GetQuiz(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	quizID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	role := c.Query("role")
	switch domain.Role(role) {
	case domain.RoleTeacher:
		h.getQuizAsTeacher(c, quizID, userID)
	case domain.RoleStudent:
		h.getQuizAsStudent(c, quizID)
	default:
		httpx.WriteError(c, apperr.ErrInvalidRole)
	}
}

func (h *Handler) getQuizAsTeacher(c *gin.Context, quizID, userID uuid.UUID) {
	detail, err := h.service.GetQuiz(c.Request.Context(), quizID, userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	questions := make([]dto.QuestionResponse, len(detail.Questions))
	for i := range detail.Questions {
		q := &detail.Questions[i]
		options := make([]dto.AnswerOptionResponse, len(q.Options))
		for j, o := range q.Options {
			options[j] = dto.AnswerOptionResponse{
				ID:        o.ID.String(),
				Text:      o.Text,
				IsCorrect: o.IsCorrect,
			}
		}
		questions[i] = dto.QuestionResponse{
			ID:         q.ID.String(),
			Type:       string(q.Type),
			Text:       q.Text,
			ImageID:    uuidPtrToString(q.ImageID),
			OrderIndex: q.OrderIndex,
			MaxScore:   q.MaxScore,
			Metadata:   q.Metadata,
			Options:    options,
		}
	}

	c.JSON(http.StatusOK, dto.QuizDetailResponse{
		ID:              detail.ID.String(),
		Title:           detail.Title,
		Description:     detail.Description,
		DefaultSettings: quizSettingsToDTO(detail.DefaultSettings),
		IsPublic:        detail.IsPublic,
		Source:          string(detail.Source),
		NeedEvaluation:  detail.NeedEvaluation,
		Questions:       questions,
	})
}

func (h *Handler) getQuizAsStudent(c *gin.Context, quizID uuid.UUID) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	view, err := h.service.GetQuizForStudent(c.Request.Context(), quizID, userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	var sessionID *string
	if view.LibraryTestSessionID != nil {
		s := view.LibraryTestSessionID.String()
		sessionID = &s
	}

	attempts := make([]dto.QuizAttemptSummaryResponse, len(view.Attempts))
	for i, a := range view.Attempts {
		var finishedAt *string
		if a.FinishedAt != nil {
			s := a.FinishedAt.Format("2006-01-02T15:04:05Z")
			finishedAt = &s
		}
		attempts[i] = dto.QuizAttemptSummaryResponse{
			ID:          a.ID.String(),
			SessionID:   a.SessionID.String(),
			SessionType: string(a.SessionType),
			Status:      string(a.Status),
			Score:       a.Score,
			StartedAt:   a.StartedAt.Format("2006-01-02T15:04:05Z"),
			FinishedAt:  finishedAt,
		}
	}

	c.JSON(http.StatusOK, dto.QuizStudentViewResponse{
		ID:          view.ID.String(),
		Title:       view.Title,
		Description: view.Description,
		DefaultSettings: dto.QuizStudentDefaultSettings{
			TotalTimeLimitSec:    view.TotalTimeLimitSec,
			QuestionTimeLimitSec: view.QuestionTimeLimitSec,
		},
		QuestionCount:        view.QuestionCount,
		LibraryTestSessionID: sessionID,
		Attempts:             attempts,
	})
}

func (h *Handler) DeleteQuiz(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	quizID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	if err := h.service.DeleteQuiz(c.Request.Context(), quizID, userID); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) UpdateQuiz(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	quizID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	var req dto.UpdateQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	var settings *domain.QuizDefaultSettings
	if req.DefaultSettings != nil {
		st, err := quizSettingsFromDTO(req.DefaultSettings)
		if err != nil {
			httpx.WriteError(c, apperr.ErrBadRequest)
			return
		}
		settings = &st
	}

	if err := h.service.UpdateQuiz(c.Request.Context(), quizID, userID, req.Title, req.Description, settings); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) PublishQuiz(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	quizID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	if err := h.service.PublishQuiz(c.Request.Context(), quizID, userID); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) CopyQuiz(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	quizID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	var req dto.CopyQuizRequest
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.WriteError(c, apperr.ErrBadRequest)
			return
		}
	}

	var attachToCourse *uuid.UUID
	if req.AttachToCourse != nil {
		cid, err := uuid.Parse(req.AttachToCourse.CourseID)
		if err != nil {
			httpx.WriteError(c, apperr.ErrBadID)
			return
		}
		attachToCourse = &cid
	}

	newID, err := h.service.CopyQuiz(c.Request.Context(), quizID, userID, attachToCourse)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CreateQuizResponse{ID: newID.String()})
}

func (h *Handler) DeleteCourseSession(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	if err := h.service.DeleteCourseSession(c.Request.Context(), sessionID, userID); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) CreateTestCourseSessionInline(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	var req dto.CreateTestCourseSessionInlineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	courseID, err := uuid.Parse(req.CourseID)
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}
	moduleID, err := uuid.Parse(req.ModuleID)
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	questions := make([]domain.AddQuestionParams, len(req.Questions))
	for i, q := range req.Questions {
		maxScore := 10
		if q.MaxScore != nil {
			maxScore = *q.MaxScore
		}
		options := make([]domain.AddOptionParams, len(q.AnswerOptions))
		for j, o := range q.AnswerOptions {
			options[j] = domain.AddOptionParams{Text: o.Text, IsCorrect: o.IsCorrect}
		}
		questions[i] = domain.AddQuestionParams{
			Type:     domain.QuestionType(q.Type),
			Text:     q.Text,
			ImageID:  parseNullableUUID(q.ImageID),
			Metadata: q.Metadata,
			MaxScore: maxScore,
			Options:  options,
		}
	}

	params := domain.CreateTestCourseSessionInlineParams{
		Title:             req.Title,
		Description:       req.Description,
		CourseID:          courseID,
		ModuleID:          moduleID,
		Questions:         questions,
		TotalTimeLimitSec: req.TotalTimeLimitSec,
		ShuffleQuestions:  req.ShuffleQuestions,
	}
	if req.StartedAt != nil {
		t, err := time.Parse(time.RFC3339, *req.StartedAt)
		if err != nil {
			httpx.WriteError(c, apperr.ErrBadRequest)
			return
		}
		params.StartedAt = &t
	}
	if req.FinishedAt != nil {
		t, err := time.Parse(time.RFC3339, *req.FinishedAt)
		if err != nil {
			httpx.WriteError(c, apperr.ErrBadRequest)
			return
		}
		params.FinishedAt = &t
	}

	quizTemplateID, sessionID, err := h.service.CreateTestCourseSessionInline(c.Request.Context(), userID, params)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CreateTestCourseSessionInlineResponse{
		QuizTemplateID: quizTemplateID.String(),
		SessionID:      sessionID.String(),
	})
}

func uuidPtrToString(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}

func parseNullableUUID(s *string) *uuid.UUID {
	if s == nil {
		return nil
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return nil
	}
	return &id
}
