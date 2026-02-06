package jwtsvc

import (
	"doorman/internal/transport/dto"
	"encoding/base64"
	"math/big"
)

type Service struct {
	keyStore KeyStore
}

func NewService(keyStore KeyStore) *Service {
	return &Service{
		keyStore: keyStore,
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
