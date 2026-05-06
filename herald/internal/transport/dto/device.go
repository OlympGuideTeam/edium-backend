package dto

type RegisterDeviceRequest struct {
	FCMToken string `json:"fcm_token" binding:"required"`
	Platform string `json:"platform" binding:"required,oneof=ios android"`
}
