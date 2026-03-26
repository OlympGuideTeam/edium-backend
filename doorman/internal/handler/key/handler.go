package keyhandler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service IKeyService
}

func NewHandler(service IKeyService) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GetJWKS(c *gin.Context) {
	resp := h.service.GetPublicKeys()

	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, resp)
}
