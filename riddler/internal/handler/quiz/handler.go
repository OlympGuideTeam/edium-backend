package quiz

import (
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

func toListItemResponse(item domain.QuizListItem) dto.QuizListItemResponse {
	return dto.QuizListItemResponse{
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
