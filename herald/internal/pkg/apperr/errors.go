package apperr

import "net/http"

var (
	ErrUnauthorized       = New("UNAUTHORIZED", "Требуется авторизация", http.StatusUnauthorized)
	ErrUnauthorizedToken  = New("UNAUTHORIZED", "Неверный или истёкший токен", http.StatusUnauthorized)
	ErrUnauthorizedClaims = New("UNAUTHORIZED", "Неверные claims", http.StatusUnauthorized)
	ErrUnauthorizedSub    = New("UNAUTHORIZED", "Неверный формат sub в токене", http.StatusUnauthorized)

	ErrBadRequest = New("BAD_REQUEST", "Некорректный запрос", http.StatusBadRequest)
	ErrBadID      = New("BAD_REQUEST", "Некорректный идентификатор", http.StatusBadRequest)

	ErrNotificationNotFound = New("NOTIFICATION_NOT_FOUND", "Уведомление не найдено", http.StatusNotFound)
	ErrDeviceNotFound       = New("DEVICE_NOT_FOUND", "Устройство не найдено", http.StatusNotFound)
)
