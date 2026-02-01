package otp

import "doorman/internal/shared/apperr"

var (
	errAlreadySent = apperr.New(
		"OTP_ALREADY_SENT",
		"Одноразовый код уже отправлен",
		429,
	)
	errPhoneUnavailable = apperr.New(
		"PHONE_UNAVAILABLE",
		"Пользователь с таким номером удален/заблокирован",
		403,
	)
)
