package regsvc

import (
	"context"
	"doorman/internal/domain"
	tokenhandler "doorman/internal/handler/token"
	"doorman/internal/pkg/correlation"
	"encoding/json"
)

type Service struct {
	identityStore IdentityStore
	regTokenStore RegTokenStore
	taskScheduler TaskScheduler
	jwtIssuer     JWTIssuer
	txManager     TxManager
}

func NewService(
	identityStore IdentityStore,
	regTokenStore RegTokenStore,
	taskScheduler TaskScheduler,
	jwtIssuer JWTIssuer,
	txManager TxManager,
) *Service {
	return &Service{
		identityStore: identityStore,
		regTokenStore: regTokenStore,
		taskScheduler: taskScheduler,
		jwtIssuer:     jwtIssuer,
		txManager:     txManager,
	}
}

type userCreatedPayload struct {
	UserID        string `json:"user_id"`
	Phone         string `json:"phone"`
	Name          string `json:"name"`
	Surname       string `json:"surname"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

func (s *Service) Register(ctx context.Context, phone, name, surname, regToken string) (*tokenhandler.AuthTokens, error) {
	storedToken, err := s.regTokenStore.Get(ctx, phone)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if storedToken != regToken {
		return nil, ErrInvalidToken
	}

	var identity domain.Identity
	err = s.txManager.WithTx(ctx, func(ctx context.Context) error {
		identity, err = s.identityStore.Create(ctx, phone)
		if err != nil {
			return err
		}

		payload, err := json.Marshal(userCreatedPayload{
			UserID:        identity.ID.String(),
			Phone:         phone,
			Name:          name,
			Surname:       surname,
			CorrelationID: correlation.IDFromContext(ctx),
		})
		if err != nil {
			return err
		}

		return s.taskScheduler.Schedule(ctx, domain.UserCreated, payload)
	})
	if err != nil {
		return nil, err
	}

	_ = s.regTokenStore.Delete(ctx, phone)

	accessToken, refreshToken, expiresIn, err := s.jwtIssuer.IssueTokens(ctx, identity.ID.String())
	if err != nil {
		return nil, err
	}

	return &tokenhandler.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    uint64(expiresIn),
	}, nil
}
