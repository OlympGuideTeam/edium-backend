package otphandler

import (
	"doorman/internal/pkg/apperr"
	"doorman/internal/pkg/httpx"
	"doorman/internal/transport/dto"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Handler struct {
	service IOTPService
}

func NewHandler(service IOTPService) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Send(c *gin.Context) {
	var req dto.SendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.BadRequest(err))
		return
	}

	err := h.service.SendOTP(c.Request.Context(), req.Phone, req.Channel)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) Verify(c *gin.Context) {
	var req dto.VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.BadRequest(err))
		return
	}

	verifyResult, err := h.service.VerifyOTP(c.Request.Context(), req.Phone, req.OTP)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	switch result := verifyResult.(type) {
	case AuthTokens:
		response := dto.AuthTokensResponse{
			AccessToken:  result.AccessToken,
			RefreshToken: result.RefreshToken,
			ExpiresIn:    result.ExpiresIn,
		}
		c.JSON(http.StatusOK, response)
	case RegistrationToken:
		response := dto.RegistrationTokenResponse{
			RegistrationToken: result.Token,
		}
		c.JSON(http.StatusOK, response)
	}
}
