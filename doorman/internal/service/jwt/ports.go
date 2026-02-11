package jwtsvc

import (
	"context"
	"crypto/rsa"
	"time"
)

type KeyStore interface {
	GetPublicKeys() map[string]*rsa.PublicKey
	GenerateAuthTokens(userID string, accessTTL time.Duration, refreshTTL time.Duration) (
		accessToken string, refreshToken string, expiresIn int64, err error,
	)
}

type RefreshTokenStore interface {
	Save(ctx context.Context, userID, token string) error
}
