package repository

import (
	"crypto/rsa"
	"crypto/x509"
	"doorman/internal/config"
	"encoding/pem"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"strings"
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

type InMemoryKeyStore struct {
	ActiveKID   string
	PrivateKeys map[string]*rsa.PrivateKey
	PublicKeys  map[string]*rsa.PublicKey
}

func NewInMemoryKeysStoreWithOneKey(config config.KeysConfig) (*InMemoryKeyStore, error) {
	if config.JwtActiveKID == "" || config.JwtRSAPrivateKey == "" {
		return nil, errors.New("empty keys config")
	}

	ks := &InMemoryKeyStore{
		ActiveKID:   config.JwtActiveKID,
		PrivateKeys: map[string]*rsa.PrivateKey{},
		PublicKeys:  map[string]*rsa.PublicKey{},
	}

	pemStr := strings.ReplaceAll(config.JwtRSAPrivateKey, `\n`, "\n")

	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("failed to decode PEM")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	var ok bool
	privateKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not an RSA private key")
	}

	ks.PrivateKeys[config.JwtActiveKID] = privateKey
	ks.PublicKeys[config.JwtActiveKID] = &privateKey.PublicKey

	return ks, nil
}

func (ks *InMemoryKeyStore) GetPublicKeys() map[string]*rsa.PublicKey {
	return ks.PublicKeys
}

func (ks *InMemoryKeyStore) GenerateAuthTokens(
	userID string,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) (accessToken string, refreshToken string, expiresIn int64, err error) {

	privateKey, ok := ks.PrivateKeys[ks.ActiveKID]
	if !ok {
		return "", "", 0, errors.New("active private key not found")
	}

	now := time.Now()

	// Access
	accessClaims := AccessClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTTL)),
		},
	}

	access := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	access.Header["kid"] = ks.ActiveKID

	accessToken, err = access.SignedString(privateKey)
	if err != nil {
		return "", "", 0, err
	}

	// Refresh
	refreshClaims := RefreshClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(refreshTTL)),
		},
	}

	refresh := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims)
	refresh.Header["kid"] = ks.ActiveKID

	refreshToken, err = refresh.SignedString(privateKey)
	if err != nil {
		return "", "", 0, err
	}

	return accessToken, refreshToken, int64(accessTTL.Seconds()), nil
}
