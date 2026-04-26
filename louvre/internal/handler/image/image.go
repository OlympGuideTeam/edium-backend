package image

import (
	"io"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"louvre/internal/pkg/apperr"
	"louvre/internal/pkg/httpx"
	"louvre/internal/service"
)

type Handler struct {
	imageService *service.ImageService
}

func NewHandler(imageService *service.ImageService) *Handler {
	return &Handler{
		imageService: imageService,
	}
}

func (h *Handler) Upload(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest.WithDetails(map[string]any{
			"reason": "параметр file обязателен",
		}))
		return
	}

	err = h.imageService.Upload(c.Request.Context(), fileHeader)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "upload image", "err", err)

	errMsg := err.Error()
	switch {
	case strings.Contains(errMsg, "файл слишком большой"):
		httpx.WriteError(c, apperr.ErrFileTooLarge.WithDetails(map[string]any{
			"reason": errMsg,
		}))
	case strings.Contains(errMsg, "невалидный тип файла"):
		httpx.WriteError(c, apperr.ErrInvalidFileType.WithDetails(map[string]any{
			"reason": errMsg,
			"allowed_types": strings.Join([]string{"image/jpeg", "image/png", "image/webp"}, ", "),
		}))
	case strings.Contains(errMsg, "превышен лимит загрузок"):
		httpx.WriteError(c, apperr.ErrTooManyUploads.WithDetails(map[string]any{
			"reason": errMsg,
			"period": "1 час",
		}))
	default:
		httpx.WriteError(c, apperr.ErrInternalError.WithDetails(map[string]any{
			"reason": errMsg,
		}))
	}
		return
	}

	c.Status(200)
}

func (h *Handler) Download(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest.WithDetails(map[string]any{
			"reason": "невалидный UUID",
		}))
		return
	}

	reader, mimeType, err := h.imageService.Download(c.Request.Context(), id)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "download image", "id", id, "err", err)
		httpx.WriteError(c, apperr.ErrNotFound)
		return
	}
	defer func() {
		if closer, ok := reader.(io.Closer); ok {
			closer.Close()
		}
	}()

	c.Header("Content-Type", mimeType)
	c.Status(200)
	if _, err := io.Copy(c.Writer, reader); err != nil {
		slog.ErrorContext(c.Request.Context(), "copy image to response", "id", id, "err", err)
	}
}

func (h *Handler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest.WithDetails(map[string]any{
			"reason": "невалидный UUID",
		}))
		return
	}

	if err := h.imageService.Delete(c.Request.Context(), id); err != nil {
		slog.ErrorContext(c.Request.Context(), "delete image", "id", id, "err", err)
		httpx.WriteError(c, apperr.ErrInternalError.WithDetails(map[string]any{
			"reason": err.Error(),
		}))
		return
	}

	c.Status(200)
}
