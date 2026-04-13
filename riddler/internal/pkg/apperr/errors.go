package apperr

import "net/http"

var (
	// Авторизация
	ErrUnauthorized       = New("UNAUTHORIZED", "Требуется авторизация", http.StatusUnauthorized)
	ErrUnauthorizedToken  = New("UNAUTHORIZED", "Неверный или истёкший токен", http.StatusUnauthorized)
	ErrUnauthorizedClaims = New("UNAUTHORIZED", "Неверные claims", http.StatusUnauthorized)
	ErrUnauthorizedSub    = New("UNAUTHORIZED", "Неверный формат sub в токене", http.StatusUnauthorized)

	// Общие
	ErrBadRequest = New("BAD_REQUEST", "Некорректный запрос", http.StatusBadRequest)
	ErrBadID      = New("BAD_REQUEST", "Некорректный идентификатор", http.StatusBadRequest)
)
