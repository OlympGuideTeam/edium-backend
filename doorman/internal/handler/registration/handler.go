package reghandler

import (
	"doorman/internal/pkg/apperr"
	"doorman/internal/pkg/httpx"
	"doorman/internal/transport/dto"
	"net/http"

	"github.com/gin-gonic/gin"
)

const regTokenHeader = "X-Reg-Token"

type Handler struct {
	service IRegistrationService
}

func NewHandler(service IRegistrationService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(c *gin.Context) {
	regToken := c.GetHeader(regTokenHeader)
	if regToken == "" {
		httpx.WriteError(c, apperr.New("MISSING_REG_TOKEN", "Отсутствует токен регистрации", http.StatusUnauthorized))
		return
	}

	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.BadRequest(err))
		return
	}

	tokens, err := h.service.Register(c.Request.Context(), req.Phone, req.Name, req.Surname, regToken)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.AuthTokensResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	})
}
