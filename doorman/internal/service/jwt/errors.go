package jwtsvc

import "doorman/internal/pkg/apperr"

var (
	ErrRefreshTokenInvalid = apperr.New(
		"REFRESH_TOKEN_INVALID",
		"Невалидный refresh токен",
		401,
	)
	ErrRefreshTokenExpired = apperr.New(
		"REFRESH_TOKEN_EXPIRED",
		"Refresh токен истёк",
		401,
	)
)
