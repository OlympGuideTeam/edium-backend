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

	var settings domain.QuizDefaultSettings
	if req.DefaultSettings != nil {
		settings = domain.QuizDefaultSettings{
			TotalTimeLimitSec:    req.DefaultSettings.TotalTimeLimitSec,
			QuestionTimeLimitSec: req.DefaultSettings.QuestionTimeLimitSec,
			ShuffleQuestions:     req.DefaultSettings.ShuffleQuestions,
		}
	}

	var attachToModule *uuid.UUID
	if req.AttachToModule != nil {
		mid, err := uuid.Parse(req.AttachToModule.ModuleID)
		if err != nil {
			httpx.WriteError(c, apperr.ErrBadID)
			return
		}
		attachToModule = &mid
	}

	id, err := h.service.CreateQuiz(c.Request.Context(), userID, req.Title, req.Description, settings, attachToModule)
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
			ImageLink:  q.ImageLink,
			OrderIndex: q.OrderIndex,
			MaxScore:   q.MaxScore,
			Metadata:   q.Metadata,
			Options:    options,
		}
	}

	c.JSON(http.StatusOK, dto.QuizDetailResponse{
		ID:          detail.ID.String(),
		Title:       detail.Title,
		Description: detail.Description,
		DefaultSettings: dto.QuizDefaultSettings{
			TotalTimeLimitSec:    detail.DefaultSettings.TotalTimeLimitSec,
			QuestionTimeLimitSec: detail.DefaultSettings.QuestionTimeLimitSec,
			ShuffleQuestions:     detail.DefaultSettings.ShuffleQuestions,
		},
		IsPublic:       detail.IsPublic,
		IsDraft:        detail.IsDraft,
		NeedEvaluation: detail.NeedEvaluation,
		Questions:      questions,
	})
}

func (h *Handler) getQuizAsStudent(c *gin.Context, quizID uuid.UUID) {
	view, err := h.service.GetQuizForStudent(c.Request.Context(), quizID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	var sessionID *string
	if view.LibraryTestSessionID != nil {
		s := view.LibraryTestSessionID.String()
		sessionID = &s
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
	})
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

	if err := h.service.UpdateQuiz(c.Request.Context(), quizID, userID, req.Title, req.Description); err != nil {
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

	var req dto.PublishQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	if err := h.service.PublishQuiz(c.Request.Context(), quizID, userID, req.IsPublic); err != nil {
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

	newID, err := h.service.CopyQuiz(c.Request.Context(), quizID, userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CreateQuizResponse{ID: newID.String()})
}

func (h *Handler) CreateTestCourseSession(c *gin.Context) {
	_, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	var req dto.CreateTestSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	quizTemplateID, err := uuid.Parse(req.QuizTemplateID)
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}
	moduleID, err := uuid.Parse(req.ModuleID)
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	params := domain.CreateTestCourseSessionParams{
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

	sessionID, err := h.service.CreateTestCourseSession(c.Request.Context(), quizTemplateID, moduleID, params)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CreateSessionResponse{SessionID: sessionID.String()})
}

func (h *Handler) CreateLiveCourseSession(c *gin.Context) {
	_, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	var req dto.CreateLiveSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	quizTemplateID, err := uuid.Parse(req.QuizTemplateID)
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}
	moduleID, err := uuid.Parse(req.ModuleID)
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	sessionID, err := h.service.CreateLiveCourseSession(c.Request.Context(), quizTemplateID, moduleID, domain.CreateLiveCourseSessionParams{
		QuestionTimeLimitSec: req.QuestionTimeLimitSec,
	})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CreateSessionResponse{SessionID: sessionID.String()})
}
