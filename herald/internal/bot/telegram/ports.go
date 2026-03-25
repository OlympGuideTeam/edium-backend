package telegram

import "context"

type OTPService interface {
	RequestOTP(ctx context.Context, chatID int64, phone string) error
}
