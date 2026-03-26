package otpsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"herald/internal/domain"
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
	Phone   string         `json:"phone"`
	Channel domain.Channel `json:"channel"`
}

func (s *Service) RequestOTP(ctx context.Context, chatID int64, phone string) error {
	payload, err := json.Marshal(otpRequestPayload{
		Phone:   phone,
		Channel: domain.ChannelTG,
	})
	if err != nil {
		return fmt.Errorf("RequestOTP marshal: %w", err)
	}

	return s.txManager.WithTx(ctx, func(ctx context.Context) error {
		if err := s.pendingOTP.Save(ctx, phone, chatID); err != nil {
			return err
		}
		return s.tasks.Schedule(ctx, domain.OTPRequest, payload)
	})
}

func (s *Service) GetPendingOTP(ctx context.Context, phone string) (*domain.PendingOTP, error) {
	return s.pendingOTP.Get(ctx, phone)
}

func (s *Service) DeletePendingOTP(ctx context.Context, phone string) error {
	return s.pendingOTP.Delete(ctx, phone)
}
