package user

import "caesar/internal/pkg/apperr"

var (
	ErrNotFound          = apperr.New("USER_NOT_FOUND", "Пользователь не найден", 404)
	ErrBlockedOrDeleted  = apperr.New("FORBIDDEN", "Пользователь заблокирован или удалён", 403)
	ErrEmptyUpdateFields = apperr.New("BAD_REQUEST", "Необходимо указать хотя бы одно поле", 400)
)
