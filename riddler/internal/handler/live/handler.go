package live

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"riddler/internal/middleware"
	"riddler/internal/pkg/apperr"
	"riddler/internal/pkg/httpx"
	"riddler/internal/transport/dto"
)

type Handler struct {
	svc liveService
}

func NewHandler(svc liveService) *Handler {
	return &Handler{svc: svc}
}

func userIDFromCtx(c *gin.Context) (uuid.UUID, bool) {
	id, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		httpx.WriteError(c, apperr.ErrUnauthorized)
		c.Abort()
	}
	return id, ok
}

func (h *Handler) JoinLiveSession(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	var req dto.JoinLiveSessionRequest
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.WriteError(c, apperr.ErrBadRequest)
			return
		}
	}

	var userID *uuid.UUID
	if id, ok := middleware.UserIDFromContext(c.Request.Context()); ok {
		userID = &id
	}

	attemptID, wsToken, err := h.svc.JoinLiveSession(c.Request.Context(), sessionID, userID, req.Name)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.JoinLiveSessionResponse{
		AttemptID: attemptID.String(),
		WsToken:   wsToken,
	})
}

func (h *Handler) ResolveLiveCode(c *gin.Context) {
	code := c.Param("code")

	meta, err := h.svc.ResolveLiveCode(c.Request.Context(), code)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ResolveLiveCodeResponse{
		SessionID:          meta.SessionID.String(),
		QuizTitle:          meta.QuizTitle,
		QuestionCount:      meta.QuestionCount,
		IsAnonymousAllowed: meta.IsAnonymousAllowed,
	})
}

func (h *Handler) StartLiveSession(c *gin.Context) {
	authorID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	wsToken, joinCode, err := h.svc.StartLiveSession(c.Request.Context(), sessionID, authorID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.StartLiveSessionResponse{WsToken: wsToken, JoinCode: joinCode})
}

func (h *Handler) CreateLiveLibrarySession(c *gin.Context) {
	authorID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	var req dto.CreateLiveLibrarySessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	quizTemplateID, err := uuid.Parse(req.QuizTemplateID)
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	sessionID, err := h.svc.CreateLiveLibrarySession(c.Request.Context(), authorID, quizTemplateID, req.QuestionTimeLimitSec)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CreateLiveSessionResponse{SessionID: sessionID.String()})
}

func (h *Handler) CreateLiveCourseSession(c *gin.Context) {
	authorID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	var req dto.CreateLiveCourseSessionRequest
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

	sessionID, err := h.svc.CreateLiveCourseSession(c.Request.Context(), authorID, quizTemplateID, moduleID, req.QuestionTimeLimitSec)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CreateLiveSessionResponse{SessionID: sessionID.String()})
}
