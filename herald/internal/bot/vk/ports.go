package vk

import (
	"context"
	"herald/internal/domain"
)

type OTPService interface {
	RequestOTP(ctx context.Context, chatID int64, phone string, channel domain.Channel) error
}
