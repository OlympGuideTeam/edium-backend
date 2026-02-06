package repository

import (
	"crypto/rsa"
	"crypto/x509"
	"doorman/internal/config"
	"encoding/pem"
	"errors"
	"strings"
)

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
