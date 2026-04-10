package apperr

import "net/http"

var (
	// Авторизация
	ErrUnauthorized       = New("UNAUTHORIZED", "Требуется авторизация", http.StatusUnauthorized)
	ErrUnauthorizedToken  = New("UNAUTHORIZED", "Неверный или истёкший токен", http.StatusUnauthorized)
	ErrUnauthorizedClaims = New("UNAUTHORIZED", "Неверные claims", http.StatusUnauthorized)
	ErrUnauthorizedSub    = New("UNAUTHORIZED", "Неверный формат sub в токене", http.StatusUnauthorized)

	// Общие
	ErrBadRequest  = New("BAD_REQUEST", "Некорректный запрос", http.StatusBadRequest)
	ErrBadID       = New("BAD_REQUEST", "Некорректный идентификатор", http.StatusBadRequest)
	ErrInvalidRole = New("BAD_REQUEST", "Параметр role обязателен: teacher или student", http.StatusBadRequest)

	// Пользователи
	ErrUserNotFound          = New("USER_NOT_FOUND", "Пользователь не найден", http.StatusNotFound)
	ErrUserBlockedOrDeleted  = New("FORBIDDEN", "Пользователь заблокирован или удалён", http.StatusForbidden)
	ErrUserEmptyUpdateFields = New("BAD_REQUEST", "Необходимо указать хотя бы одно поле", http.StatusBadRequest)

	// Классы
	ErrClassNotFound      = New("CLASS_NOT_FOUND", "Класс не найден", http.StatusNotFound)
	ErrClassForbidden     = New("FORBIDDEN", "Только владелец класса может выполнить это действие", http.StatusForbidden)
	ErrClassRemoveOwner   = New("FORBIDDEN", "Нельзя исключить владельца класса", http.StatusForbidden)
	ErrClassEmptyTitle    = New("VALIDATION_ERROR", "Название класса не может быть пустым", http.StatusUnprocessableEntity)
	ErrInvitationNotFound = New("INVITATION_NOT_FOUND", "Приглашение не найдено", http.StatusNotFound)
	ErrAlreadyMember      = New("ALREADY_MEMBER", "Пользователь уже является участником класса", http.StatusConflict)
	ErrMemberNotFound     = New("MEMBER_NOT_FOUND", "Участник не найден в классе", http.StatusNotFound)

	// Курсы
	ErrCourseNotFound   = New("COURSE_NOT_FOUND", "Курс не найден", http.StatusNotFound)
	ErrModuleNotFound   = New("MODULE_NOT_FOUND", "Модуль не найден", http.StatusNotFound)
	ErrCourseForbidden  = New("FORBIDDEN", "Нет прав для выполнения этого действия", http.StatusForbidden)
	ErrCourseEmptyTitle = New("VALIDATION_ERROR", "Название не может быть пустым", http.StatusUnprocessableEntity)
	ErrCourseNotMember  = New("FORBIDDEN", "Вы не являетесь участником этого класса", http.StatusForbidden)
)
