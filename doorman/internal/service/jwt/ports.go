package jwtsvc

import "crypto/rsa"

type KeyStore interface {
	GetPublicKeys() map[string]*rsa.PublicKey
}
