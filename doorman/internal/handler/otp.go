package handler

import (
	"doorman/internal/pkg/apperr"
	"doorman/internal/pkg/httpx"
	"doorman/internal/transport/dto"
	"github.com/gin-gonic/gin"
	"net/http"
)

type OTPHandler struct {
	service IOTPService
}

func NewHandler(service IOTPService) *OTPHandler {
	return &OTPHandler{
		service: service,
	}
}

func (h *OTPHandler) Send(c *gin.Context) {
	var req dto.SendOTPRequest
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

func (h *OTPHandler) Verify(c *gin.Context) {

}
