package otpsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"herald/internal/domain"
	"herald/internal/pkg/correlation"
	"log"

	"github.com/google/uuid"
)

type Service struct {
	txManager  TxManager
	tasks      TaskRepository
	pendingOTP PendingOTPRepository
}

func NewService(txManager TxManager, tasks TaskRepository, pendingOTP PendingOTPRepository) *Service {
	return &Service{
		txManager:  txManager,
		tasks:      tasks,
		pendingOTP: pendingOTP,
	}
}

type otpRequestPayload struct {
	Phone         string         `json:"phone"`
	Channel       domain.Channel `json:"channel"`
	CorrelationID string         `json:"correlation_id"`
}

type otpDeliveryPayload struct {
	ChatID        int64  `json:"chat_id"`
	OTP           uint64 `json:"otp"`
	CorrelationID string `json:"correlation_id"`
}

func (s *Service) RequestOTP(ctx context.Context, chatID int64, phone string) error {
	correlationID := correlation.IDFromContext(ctx)
	if correlationID == "" {
		correlationID = uuid.New().String()
		ctx = correlation.WithID(ctx, correlationID)
	}

	payload, err := json.Marshal(otpRequestPayload{
		Phone:         phone,
		Channel:       domain.ChannelTG,
		CorrelationID: correlationID,
	})
	if err != nil {
		return fmt.Errorf("RequestOTP marshal: %w", err)
	}

	return s.txManager.WithTx(ctx, func(ctx context.Context) error {
		if err := s.pendingOTP.Save(ctx, correlationID, chatID); err != nil {
			return err
		}
		return s.tasks.Schedule(ctx, domain.OTPRequest, payload)
	})
}

func (s *Service) HandleOTPSent(ctx context.Context, correlationID string, otp uint64) error {
	return s.txManager.WithTx(ctx, func(ctx context.Context) error {
		pending, err := s.pendingOTP.Get(ctx, correlationID)
		if err != nil {
			return err
		}
		if pending == nil {
			log.Printf("[otp-svc] pending_otp not found or expired, correlation_id=%s", correlationID)
			return nil
		}

		deliveryPayload, err := json.Marshal(otpDeliveryPayload{
			ChatID:        pending.ChatID,
			OTP:           otp,
			CorrelationID: correlationID,
		})
		if err != nil {
			return fmt.Errorf("HandleOTPSent marshal: %w", err)
		}

		if err := s.tasks.Schedule(ctx, domain.OTPDelivery, deliveryPayload); err != nil {
			return err
		}
		return s.pendingOTP.Delete(ctx, correlationID)
	})
}
