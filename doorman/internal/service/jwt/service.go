package jwtsvc

import (
	"context"
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
	accessToken, refreshToken, expiresIn, err := s.keyStore.GenerateAuthTokens(
		userID,
		accessTTL,
		refreshTTL,
	)
	if err != nil {
		return "", "", 0, err
	}

	err = s.refreshTokenStore.Save(ctx, userID, refreshToken)
	if err != nil {
		return "", "", 0, err
	}

	return accessToken, refreshToken, expiresIn, nil
}
