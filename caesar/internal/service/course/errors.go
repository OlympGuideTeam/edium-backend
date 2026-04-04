package course

import "caesar/internal/pkg/apperr"

var (
	ErrNotFound       = apperr.New("COURSE_NOT_FOUND", "Курс не найден", 404)
	ErrModuleNotFound = apperr.New("MODULE_NOT_FOUND", "Модуль не найден", 404)
	ErrForbidden      = apperr.New("FORBIDDEN", "Нет прав для выполнения этого действия", 403)
	ErrEmptyTitle     = apperr.New("VALIDATION_ERROR", "Название не может быть пустым", 422)
	ErrNotMember      = apperr.New("FORBIDDEN", "Вы не являетесь участником этого класса", 403)
)
