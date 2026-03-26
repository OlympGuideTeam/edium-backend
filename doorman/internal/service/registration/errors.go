package regsvc

import (
	"doorman/internal/pkg/apperr"
	"net/http"
)

var (
	ErrInvalidToken = apperr.New("INVALID_REG_TOKEN", "Недействительный токен регистрации", http.StatusUnauthorized)
)
