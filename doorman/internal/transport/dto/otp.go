package dto

import "doorman/internal/domain"

type SendOTPRequest struct {
	Phone   string         `json:"phone" binding:"required,e164,startswith=+7"`
	Channel domain.Channel `json:"channel" binding:"required,oneof=tg max"`
}
