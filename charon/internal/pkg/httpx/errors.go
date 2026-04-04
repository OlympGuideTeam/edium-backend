package httpx

import (
	"errors"
	"net/http"

	"charon/internal/pkg/apperr"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Error       string         `json:"error"`
	Description string         `json:"description"`
	Details     map[string]any `json:"details,omitempty"`
}

func WriteError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		c.JSON(appErr.HTTPStatus, ErrorResponse{
			Error:       appErr.Code,
			Description: appErr.Description,
			Details:     appErr.Details,
		})
		return
	}

	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error:       "INTERNAL_ERROR",
		Description: "Internal error",
		Details: map[string]any{
			"error": err.Error(),
		},
	})
}
