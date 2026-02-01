package otp

import (
	"doorman/internal/shared/apperr"
	"doorman/internal/shared/httpx"
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
	var req sendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.BadRequest(err))
		return
	}

	err := h.service.SendOTP(c.Request.Context(), req.Phone, req.Channel)
	if err != nil {
		httpx.WriteError(c, err)
	}

	c.Status(http.StatusOK)
}

func (h *Handler) Verify(c *gin.Context) {

}
