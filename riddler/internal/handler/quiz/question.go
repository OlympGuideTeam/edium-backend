package quiz

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/pkg/apperr"
	"riddler/internal/pkg/httpx"
	"riddler/internal/transport/dto"
)

func (h *Handler) AddQuestion(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	quizID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	var req dto.AddQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	maxScore := 10
	if req.MaxScore != nil {
		maxScore = *req.MaxScore
	}

	options := make([]domain.AddOptionParams, len(req.AnswerOptions))
	for i, o := range req.AnswerOptions {
		options[i] = domain.AddOptionParams{Text: o.Text, IsCorrect: o.IsCorrect}
	}

	params := domain.AddQuestionParams{
		Type:     domain.QuestionType(req.Type),
		Text:     req.Text,
		ImageID:  parseNullableUUID(req.ImageID),
		Metadata: req.Metadata,
		MaxScore: maxScore,
		Options:  options,
	}

	id, orderIndex, err := h.service.AddQuestion(c.Request.Context(), quizID, userID, params)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.AddQuestionResponse{ID: id.String(), OrderIndex: orderIndex})
}

func (h *Handler) DeleteQuestion(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	quizID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	questionID, err := uuid.Parse(c.Param("question_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	if err := h.service.DeleteQuestion(c.Request.Context(), quizID, questionID, userID); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) GenerateQuestions(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	quizID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	var req dto.GenerateQuestionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	jobID, err := h.service.GenerateQuestions(c.Request.Context(), quizID, userID, req.Text)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.GenerateQuestionsResponse{JobID: jobID.String()})
}

func (h *Handler) ReorderQuestions(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	quizID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	var req dto.ReorderQuestionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	questionIDs := make([]uuid.UUID, 0, len(req.QuestionIDs))
	for _, raw := range req.QuestionIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(c, apperr.ErrBadID)
			return
		}
		questionIDs = append(questionIDs, id)
	}

	if err := h.service.ReorderQuestions(c.Request.Context(), quizID, userID, questionIDs); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusOK)
}
