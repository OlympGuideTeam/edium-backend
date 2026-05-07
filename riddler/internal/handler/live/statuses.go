package live

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"riddler/internal/pkg/apperr"
	"riddler/internal/pkg/httpx"
	"riddler/internal/transport/dto"
)

const maxSessionStatusIDs = 50

func (h *Handler) GetSessionStatuses(c *gin.Context) {
	raw := c.Query("ids")
	if raw == "" {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	parts := strings.Split(raw, ",")
	if len(parts) > maxSessionStatusIDs {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	ids := make([]uuid.UUID, 0, len(parts))
	for _, p := range parts {
		id, err := uuid.Parse(strings.TrimSpace(p))
		if err != nil {
			httpx.WriteError(c, apperr.ErrBadID)
			return
		}
		ids = append(ids, id)
	}

	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	items, err := h.svc.GetSessionStatuses(c.Request.Context(), ids, userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	resp := dto.GetSessionStatusesResponse{
		Items: make([]dto.SessionStatusItemDTO, 0, len(items)),
	}
	for _, it := range items {
		item := dto.SessionStatusItemDTO{
			SessionID: it.SessionID.String(),
			Mode:      string(it.Mode),
			Status:    string(it.Status),
		}
		if it.Phase != nil {
			s := string(*it.Phase)
			item.Phase = &s
		}
		if it.AttemptStatus != nil {
			s := string(*it.AttemptStatus)
			item.AttemptStatus = &s
		}
		item.Score = it.Score
		resp.Items = append(resp.Items, item)
	}

	c.JSON(http.StatusOK, resp)
}
