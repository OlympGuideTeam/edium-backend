package otp

import (
	"context"
	"doorman/internal/domain"

	"math/rand"
)

type Service struct {
	identityStore IIdentityStore
	taskStore     ITaskStore
	otpStore      IOTPStore
}

func NewService(identityStore IIdentityStore, taskStore ITaskStore, otpStore IOTPStore) *Service {
	return &Service{
		identityStore: identityStore,
		taskStore:     taskStore,
		otpStore:      otpStore,
	}
}

func (s *Service) SendOTP(context context.Context, phone string, channel channel) error {
	identity, err := s.identityStore.GetByPhone(context, phone)
	if err != nil { // TODO: кроме ошибки, что пользователь не существует
		return err
	}

	if identity.Status == domain.IdentityStatusBlocked || identity.Status == domain.IdentityStatusDeleted {
		return errPhoneUnavailable
	}

	otp := rand.Int63n(900000) + 100000

	err = s.otpStore.Save(context, phone, otp)
	if err != nil {
		return err
	}

	return s.taskStore.EnqueueOTP(context, phone, otp, channel)
}
