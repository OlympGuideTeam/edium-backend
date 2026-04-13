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

	// Квизы
	ErrQuizNotFound   = New("QUIZ_NOT_FOUND", "Квиз не найден", http.StatusNotFound)
	ErrQuizForbidden  = New("FORBIDDEN", "Нет прав для выполнения этого действия", http.StatusForbidden)
	ErrQuizEmptyTitle = New("VALIDATION_ERROR", "Название квиза не может быть пустым", http.StatusUnprocessableEntity)

	// Вопросы
	ErrQuestionNotFound = New("QUESTION_NOT_FOUND", "Вопрос не найден", http.StatusNotFound)
)
