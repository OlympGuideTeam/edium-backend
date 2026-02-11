package keyhandler

import "doorman/internal/transport/dto"

type IKeyService interface {
	GetPublicKeys() dto.JWKSResponse
}
