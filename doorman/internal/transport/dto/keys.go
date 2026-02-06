package dto

type JWKResponse struct {
	KTy string `json:"kty"`
	KID string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type JWKSResponse struct {
	Keys []JWKResponse `json:"keys"`
}
