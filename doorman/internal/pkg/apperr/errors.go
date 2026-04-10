package apperr

import "net/http"

var (
	// OTP
	ErrPhoneUnavailable     = New("PHONE_UNAVAILABLE", "Пользователь с таким номером удален/заблокирован", http.StatusForbidden)
	ErrOTPAlreadySent       = New("OTP_ALREADY_SENT", "Одноразовый код уже отправлен", http.StatusTooManyRequests)
	ErrOTPDailyLimit        = New("OTP_DAILY_LIMIT_EXCEEDED", "Превышен дневной лимит отправки кодов", http.StatusTooManyRequests)
	ErrOTPNotFoundOrExpired = New("OTP_NOT_FOUND_OR_EXPIRED", "Одноразовый код для данного номера не существует или истёк", http.StatusBadRequest)
	ErrOTPInvalid           = New("OTP_INVALID", "Неверный код", http.StatusBadRequest)
	ErrOTPAttemptsExceeded  = New("OTP_ATTEMPTS_EXCEEDED", "Слишком много попыток", http.StatusTooManyRequests)

	// JWT
	ErrRefreshTokenInvalid = New("REFRESH_TOKEN_INVALID", "Невалидный refresh токен", http.StatusUnauthorized)
	ErrRefreshTokenExpired = New("REFRESH_TOKEN_EXPIRED", "Refresh токен истёк", http.StatusUnauthorized)

	// Регистрация
	ErrRegTokenMissing = New("MISSING_REG_TOKEN", "Отсутствует токен регистрации", http.StatusUnauthorized)
	ErrRegTokenInvalid = New("INVALID_REG_TOKEN", "Недействительный токен регистрации", http.StatusUnauthorized)
)
