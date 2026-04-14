package attempt

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"riddler/internal/pkg/apperr"
	"riddler/internal/pkg/httpx"
	"riddler/internal/transport/dto"
)

func (h *Handler) CreateAttempt(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	attempt, questions, err := h.service.Create(c.Request.Context(), sessionID, userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.CreateAttemptResponse{
		AttemptID: attempt.ID.String(),
		Questions: toQuestionsResponse(questions),
	})
}

func (h *Handler) SubmitAnswer(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	attemptID, err := uuid.Parse(c.Param("attempt_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	var req dto.SubmitAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	questionID, err := uuid.Parse(req.QuestionID)
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	if err := h.service.SubmitAnswer(c.Request.Context(), attemptID, userID, questionID, req.AnswerData); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) Finish(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	attemptID, err := uuid.Parse(c.Param("attempt_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	if err := h.service.Finish(c.Request.Context(), attemptID, userID); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) GetResult(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	attemptID, err := uuid.Parse(c.Param("attempt_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	result, err := h.service.GetResult(c.Request.Context(), attemptID, userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, toAttemptResultResponse(result))
}
