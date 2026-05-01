package test

import (
	"github.com/google/uuid"

	"github.com/gin-gonic/gin"

	"riddler/internal/middleware"
	"riddler/internal/pkg/apperr"
	"riddler/internal/pkg/httpx"
)

type Handler struct {
	svc testService
}

func NewHandler(svc testService) *Handler {
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
