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
	ErrInvalidRole = New("BAD_REQUEST", "Некорректная роль, допустимые значения: teacher, student", http.StatusBadRequest)

	// Квизы
	ErrQuizNotFound     = New("QUIZ_NOT_FOUND", "Квиз не найден", http.StatusNotFound)
	ErrQuizForbidden    = New("FORBIDDEN", "Нет прав для выполнения этого действия", http.StatusForbidden)
	ErrQuizEmptyTitle   = New("VALIDATION_ERROR", "Название квиза не может быть пустым", http.StatusUnprocessableEntity)
	ErrQuizNotPublished = New("QUIZ_NOT_PUBLISHED", "Копировать можно только опубликованный квиз", http.StatusUnprocessableEntity)

	// Вопросы
	ErrQuestionNotFound        = New("QUESTION_NOT_FOUND", "Вопрос не найден", http.StatusNotFound)
	ErrQuestionInvalidType     = New("VALIDATION_ERROR", "Неизвестный тип вопроса", http.StatusUnprocessableEntity)
	ErrQuestionOptionsRequired = New("VALIDATION_ERROR", "Необходимо указать варианты ответа", http.StatusUnprocessableEntity)
	ErrQuestionOneCorrect      = New("VALIDATION_ERROR", "Для single_choice должен быть ровно один правильный ответ", http.StatusUnprocessableEntity)
	ErrQuestionNoCorrect       = New("VALIDATION_ERROR", "Необходимо указать хотя бы один правильный ответ", http.StatusUnprocessableEntity)
	ErrQuestionMetadataInvalid = New("VALIDATION_ERROR", "Некорректная структура metadata для данного типа вопроса", http.StatusUnprocessableEntity)
)
