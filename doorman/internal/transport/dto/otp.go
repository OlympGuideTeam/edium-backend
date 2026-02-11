package dto

import "doorman/internal/domain"

type SendOTPRequest struct {
	Phone   string         `json:"phone" binding:"required,e164,startswith=+7"`
	Channel domain.Channel `json:"channel" binding:"required,oneof=tg max"`
}

type VerifyOTPRequest struct {
	Phone string `json:"phone" binding:"required,e164,startswith=+7"`
	OTP   uint64 `json:"otp" binding:"required,lt=1000000"`
}

type AuthTokensResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    uint64 `json:"expires_in"`
}

type RegistrationTokenResponse struct {
	RegistrationToken string `json:"registration_token"`
}
