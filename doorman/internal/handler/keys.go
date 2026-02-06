package handler

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type KeysHandler struct {
	service IKeyService
}

func NewKeysHandler(service IKeyService) *KeysHandler {
	return &KeysHandler{
		service: service,
	}
}

func (h *KeysHandler) GetJWKS(c *gin.Context) {
	resp := h.service.GetPublicKeys()

	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, resp)
}
