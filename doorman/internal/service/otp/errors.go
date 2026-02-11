package otpsvc

import "doorman/internal/pkg/apperr"

var (
	// Common
	ErrPhoneUnavailable = apperr.New(
		"PHONE_UNAVAILABLE",
		"Пользователь с таким номером удален/заблокирован",
		403,
	)

	// Send
	ErrAlreadySent = apperr.New(
		"OTP_ALREADY_SENT",
		"Одноразовый код уже отправлен",
		429,
	)

	// Verify
	ErrNotFoundOrExpired = apperr.New(
		"OTP_NOT_FOUND_OR_EXPIRED",
		"Одноразовый код для данного номера не существует или истёк",
		400,
	)
	ErrInvalid = apperr.New(
		"OTP_INVALID",
		"Неверный код",
		400,
	)
	ErrAttemptsExceeded = apperr.New(
		"OTP_ATTEMPTS_EXCEEDED",
		"Слишком много попыток",
		429,
	)
)
