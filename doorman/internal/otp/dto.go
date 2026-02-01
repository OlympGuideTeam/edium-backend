package otp

type channel string

const (
	channelTg  channel = "tg"
	channelMax channel = "max"
)

type sendOTPRequest struct {
	Phone   string  `json:"phone" binding:"required,e164,startswith=+7"`
	Channel channel `json:"channel" binding:"required,oneof=tg max"`
}
