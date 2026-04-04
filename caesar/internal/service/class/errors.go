package class

import "caesar/internal/pkg/apperr"

var (
	ErrNotFound    = apperr.New("CLASS_NOT_FOUND", "Класс не найден", 404)
	ErrForbidden   = apperr.New("FORBIDDEN", "Только владелец класса может выполнить это действие", 403)
	ErrRemoveOwner = apperr.New("FORBIDDEN", "Нельзя исключить владельца класса", 403)
	ErrEmptyTitle  = apperr.New("VALIDATION_ERROR", "Название класса не может быть пустым", 422)
)
