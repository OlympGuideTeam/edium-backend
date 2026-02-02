package otpsvc

import "doorman/internal/pkg/apperr"

var (
	ErrAlreadySent = apperr.New(
		"OTP_ALREADY_SENT",
		"Одноразовый код уже отправлен",
		429,
	)
	ErrPhoneUnavailable = apperr.New(
		"PHONE_UNAVAILABLE",
		"Пользователь с таким номером удален/заблокирован",
		403,
	)
)
