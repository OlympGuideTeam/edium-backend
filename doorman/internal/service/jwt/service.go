package jwtsvc

import (
	"context"
	tokenhandler "doorman/internal/handler/token"
	"doorman/internal/transport/dto"
	"encoding/base64"
	"math/big"
	"time"
)

const (
	accessTTL  = 15 * time.Minute
	refreshTTL = 30 * 24 * time.Hour
)

type Service struct {
	keyStore          KeyStore
	refreshTokenStore RefreshTokenStore
}

func NewService(keyStore KeyStore, refreshTokenStore RefreshTokenStore) *Service {
	return &Service{
		keyStore:          keyStore,
		refreshTokenStore: refreshTokenStore,
	}
}

func (s *Service) GetPublicKeys() dto.JWKSResponse {
	var keysResponse []dto.JWKResponse

	for keyID, pub := range s.keyStore.GetPublicKeys() {
		keysResponse = append(keysResponse, dto.JWKResponse{
			KTy: "RSA",
			KID: keyID,
			Use: "sig",
			Alg: "RS256",
			N:   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		})
	}

	return dto.JWKSResponse{
		Keys: keysResponse,
	}
}

func (s *Service) IssueTokens(ctx context.Context, userID string) (string, string, int64, error) {
	authTokens, err := s.keyStore.GenerateAuthTokens(
		userID,
		accessTTL,
		refreshTTL,
	)
	if err != nil {
		return "", "", 0, err
	}

	err = s.refreshTokenStore.SaveToken(ctx, authTokens.RefreshJti, userID, refreshTTL)
	if err != nil {
		return "", "", 0, err
	}

	return authTokens.AccessToken, authTokens.RefreshToken, authTokens.ExpiresIn, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	claims, err := s.keyStore.ParseRefreshToken(refreshToken)
	if err != nil {
		return err
	}

	userID, err := s.refreshTokenStore.GetAndDelToken(ctx, claims.ID)
	if err != nil {
		return err
	}

	if userID != claims.Subject {
		return ErrRefreshTokenInvalid
	}

	return nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*tokenhandler.AuthTokens, error) {
	claims, err := s.keyStore.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	userID, err := s.refreshTokenStore.GetAndDelToken(ctx, claims.ID)
	if err != nil {
		return nil, err
	}

	if userID != claims.Subject {
		return nil, ErrRefreshTokenInvalid
	}

	accessToken, refreshToken, expiresIn, err := s.IssueTokens(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &tokenhandler.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    uint64(expiresIn),
	}, nil
}
