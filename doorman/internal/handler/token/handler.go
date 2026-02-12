package tokenhandler

import (
	"doorman/internal/pkg/apperr"
	"doorman/internal/pkg/httpx"
	"doorman/internal/transport/dto"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Handler struct {
	service ITokenService
}

func NewHandler(service ITokenService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.BadRequest(err))
		return
	}

	tokens, err := h.service.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	response := dto.AuthTokensResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) Logout(c *gin.Context) {
	var req dto.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.BadRequest(err))
		return
	}

	err := h.service.Logout(c.Request.Context(), req.RefreshToken)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
