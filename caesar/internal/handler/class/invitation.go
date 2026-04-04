package class

import (
	"net/http"

	"caesar/internal/domain"
	"caesar/internal/pkg/apperr"
	"caesar/internal/pkg/httpx"
	"caesar/internal/transport/dto"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) GetInviteLink(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	classID, ok := parseClassID(c)
	if !ok {
		return
	}

	roleParam := c.Query("role")
	var role domain.ClassMemberRole
	switch roleParam {
	case string(domain.ClassMemberRoleTeacher):
		role = domain.ClassMemberRoleTeacher
	case string(domain.ClassMemberRoleStudent):
		role = domain.ClassMemberRoleStudent
	default:
		httpx.WriteError(c, apperr.ErrInvalidRole)
		return
	}

	invitationID, err := h.service.GetInviteLink(c.Request.Context(), classID, userID, role)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.InviteResponse{InvitationID: invitationID.String()})
}

func (h *Handler) AcceptInvitation(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	invitationID, err := uuid.Parse(c.Param("invitationId"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	if err := h.service.AcceptInvitation(c.Request.Context(), invitationID, userID); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
