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
		},
		IsPublic:       detail.IsPublic,
		IsDraft:        detail.IsDraft,
		NeedEvaluation: detail.NeedEvaluation,
		Questions:      questions,
	})
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

func (h *Handler) ListQuizzes(c *gin.Context) {
	role := domain.Role(c.Query("role"))
	if role != domain.RoleTeacher && role != domain.RoleStudent {
		httpx.WriteError(c, apperr.ErrInvalidRole)
		return
	}

	items, err := h.service.ListQuizzes(c.Request.Context(), role)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	result := make([]dto.QuizListItemResponse, len(items))
	for i, item := range items {
		result[i] = dto.QuizListItemResponse{
			ID:          item.ID.String(),
			Title:       item.Title,
			Description: item.Description,
			DefaultSettings: dto.QuizDefaultSettings{
				TotalTimeLimitSec:    item.DefaultSettings.TotalTimeLimitSec,
				QuestionTimeLimitSec: item.DefaultSettings.QuestionTimeLimitSec,
			},
			IsPublic:       item.IsPublic,
			IsDraft:        item.IsDraft,
			NeedEvaluation: item.NeedEvaluation,
			QuestionCount:  item.QuestionCount,
		}
	}

	c.JSON(http.StatusOK, result)
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

func (h *Handler) ListMyQuizzes(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	items, err := h.service.ListMyQuizzes(c.Request.Context(), userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	result := make([]dto.QuizListItemResponse, len(items))
	for i, item := range items {
		result[i] = dto.QuizListItemResponse{
			ID:          item.ID.String(),
			Title:       item.Title,
			Description: item.Description,
			DefaultSettings: dto.QuizDefaultSettings{
				TotalTimeLimitSec:    item.DefaultSettings.TotalTimeLimitSec,
				QuestionTimeLimitSec: item.DefaultSettings.QuestionTimeLimitSec,
			},
			IsPublic:       item.IsPublic,
			IsDraft:        item.IsDraft,
			NeedEvaluation: item.NeedEvaluation,
			QuestionCount:  item.QuestionCount,
		}
	}

	c.JSON(http.StatusOK, result)
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
