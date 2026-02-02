package otpsvc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"doorman/internal/domain"
	"doorman/internal/service"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"time"
)

type Service struct {
	identityStore IdentityStore
	taskScheduler TaskScheduler
	otpStore      OTPStore
	txManager     service.ITxManager
}

func NewService(
	txManager service.ITxManager, identityStore IdentityStore, taskScheduler TaskScheduler, otpStore OTPStore,
) *Service {
	return &Service{
		identityStore: identityStore,
		taskScheduler: taskScheduler,
		otpStore:      otpStore,
		txManager:     txManager,
	}
}

func (s *Service) SendOTP(ctx context.Context, phone string, channel domain.Channel) error {
	return s.txManager.WithTx(ctx, func(ctx context.Context) error {
		exists, err := s.otpStore.Exists(ctx, phone)
		if err != nil {
			return err
		}

		if exists {
			return ErrAlreadySent
		}

		identity, err := s.identityStore.GetByPhone(ctx, phone)
		if err != nil {
			return err
		}

		if identity != nil &&
			(identity.Status == domain.IdentityStatusBlocked || identity.Status == domain.IdentityStatusDeleted) {
			return ErrPhoneUnavailable
		}

		otp, err := s.generateOTP()
		if err != nil {
			return err
		}

		hashOTP := s.hashOTP(otp)

		err = s.otpStore.Save(ctx, phone, hashOTP, 3*time.Minute)
		if err != nil {
			return err
		}

		payload, err := json.Marshal(struct {
			Phone   string         `json:"phone"`
			OTP     uint64         `json:"otp"`
			Channel domain.Channel `json:"channel"`
		}{
			Phone:   phone,
			OTP:     otp,
			Channel: channel,
		})
		if err != nil {
			return err
		}

		return s.taskScheduler.Schedule(ctx, domain.OTPSent, payload)
	})
}

func (s *Service) generateOTP() (uint64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return 0, err
	}
	return n.Uint64() + 100000, nil
}

func (s *Service) hashOTP(otp uint64) string {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, otp)

	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
}
