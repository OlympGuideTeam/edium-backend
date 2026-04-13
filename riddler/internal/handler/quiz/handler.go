package quiz

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/middleware"
	"riddler/internal/pkg/apperr"
	"riddler/internal/pkg/httpx"
	"riddler/internal/transport/dto"
)

type Handler struct {
	service quizService
}

func NewHandler(service quizService) *Handler {
	return &Handler{service: service}
}

func userIDFromCtx(c *gin.Context) (uuid.UUID, bool) {
	id, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		httpx.WriteError(c, apperr.ErrUnauthorized)
		c.Abort()
	}
	return id, ok
}

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
		}
	}

	id, err := h.service.CreateQuiz(c.Request.Context(), userID, req.Title, req.Description, settings)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CreateQuizResponse{ID: id.String()})
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
		Type:      domain.QuestionType(req.Type),
		Text:      req.Text,
		ImageLink: req.ImageLink,
		Metadata:  req.Metadata,
		MaxScore:  maxScore,
		Options:   options,
	}

	id, orderIndex, err := h.service.AddQuestion(c.Request.Context(), quizID, userID, params)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.AddQuestionResponse{ID: id.String(), OrderIndex: orderIndex})
}
