package jwtsvc

import (
	"context"
	"crypto/rsa"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type AccessClaims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

type AuthTokensData struct {
	AccessToken  string
	RefreshToken string
	RefreshJti   string
	ExpiresIn    int64
}

type KeyStore interface {
	GetPublicKeys() map[string]*rsa.PublicKey
	GenerateAuthTokens(userID string, accessTTL time.Duration, refreshTTL time.Duration) (*AuthTokensData, error)
	ParseRefreshToken(tokenString string) (*RefreshClaims, error)
}

type RefreshTokenStore interface {
	SaveToken(ctx context.Context, jti string, userID string, ttl time.Duration) error
	GetAndDelToken(ctx context.Context, jti string) (string, error)
}
