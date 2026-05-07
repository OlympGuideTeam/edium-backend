package test

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

func (h *Handler) FinishTestCourseSession(c *gin.Context) {
	teacherID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	if err := h.svc.FinishTestCourseSession(c.Request.Context(), teacherID, sessionID); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) CreateTestCourseSession(c *gin.Context) {
	authorID, ok := userIDFromCtx(c)
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

	sessionID, err := h.svc.CreateTestCourseSession(c.Request.Context(), authorID, quizTemplateID, moduleID, params)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CreateSessionResponse{SessionID: sessionID.String()})
}
