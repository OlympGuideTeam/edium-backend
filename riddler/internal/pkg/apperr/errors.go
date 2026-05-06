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
	ErrQuizNotFound         = New("QUIZ_NOT_FOUND", "Квиз не найден", http.StatusNotFound)
	ErrQuizForbidden        = New("FORBIDDEN", "Нет прав для выполнения этого действия", http.StatusForbidden)
	ErrQuizEmptyTitle       = New("VALIDATION_ERROR", "Название квиза не может быть пустым", http.StatusUnprocessableEntity)
	ErrQuizNotPublished     = New("QUIZ_NOT_PUBLISHED", "Копировать можно только опубликованный квиз", http.StatusUnprocessableEntity)
	ErrQuizAlreadyPublished = New("QUIZ_ALREADY_PUBLISHED", "Квиз уже опубликован", http.StatusConflict)
	ErrQuizCourseOnly       = New("QUIZ_COURSE_ONLY", "Квиз курса нельзя опубликовать в библиотеку", http.StatusUnprocessableEntity)
	ErrQuizNotAvailable     = New("QUIZ_NOT_AVAILABLE", "Квиз недоступен", http.StatusForbidden)
	ErrQuizIsPublic         = New("QUIZ_IS_PUBLIC", "Нельзя удалить опубликованный квиз", http.StatusConflict)
	ErrQuizHasSessions      = New("QUIZ_HAS_SESSIONS", "Нельзя удалить квиз с существующими сессиями", http.StatusConflict)

	// Сессии
	ErrSessionNotFound       = New("SESSION_NOT_FOUND", "Сессия не найдена", http.StatusNotFound)
	ErrSessionNotActive      = New("SESSION_NOT_ACTIVE", "Сессия недоступна", http.StatusConflict)
	ErrSessionNotStarted     = New("SESSION_NOT_STARTED", "Сессия ещё не началась", http.StatusConflict)
	ErrSessionDeadlinePassed = New("SESSION_DEADLINE_PASSED", "Дедлайн сессии истёк", http.StatusConflict)
	ErrSessionAlreadyStarted = New("SESSION_ALREADY_STARTED", "Сессия уже запущена", http.StatusConflict)
	ErrSessionHasAttempts    = New("SESSION_HAS_ATTEMPTS", "Нельзя удалить сессию с попытками", http.StatusConflict)
	ErrSessionCompleted      = New("SESSION_COMPLETED", "Сессия уже завершена", http.StatusConflict)

	// Попытки
	ErrAttemptNotFound     = New("ATTEMPT_NOT_FOUND", "Попытка не найдена", http.StatusNotFound)
	ErrAttemptForbidden    = New("FORBIDDEN", "Нет доступа к попытке", http.StatusForbidden)
	ErrAttemptNotActive    = New("ATTEMPT_NOT_ACTIVE", "Попытка уже завершена", http.StatusConflict)
	ErrAttemptExpired      = New("ATTEMPT_EXPIRED", "Время попытки истекло", http.StatusConflict)
	ErrAttemptNotGraded    = New("ATTEMPT_NOT_GRADED", "Попытка ещё не проверена", http.StatusConflict)
	ErrAttemptNotCompleted = New("ATTEMPT_NOT_COMPLETED", "Не все попытки завершены", http.StatusConflict)
	ErrAttemptNotAllGraded = New("ATTEMPT_NOT_ALL_GRADED", "Не все свободные ответы оценены", http.StatusConflict)

	// Ответы
	ErrSubmissionNotFound = New("SUBMISSION_NOT_FOUND", "Ответ не найден", http.StatusNotFound)
	ErrScoreInvalid       = New("VALIDATION_ERROR", "Оценка должна быть неотрицательной", http.StatusUnprocessableEntity)

	// Live-сессии
	ErrLiveTemplateInvalid = New("LIVE_TEMPLATE_INVALID", "Шаблон не подходит для лайва: содержит with_free_answer или need_evaluation=true", http.StatusUnprocessableEntity)
	ErrLiveNotInLobby      = New("LIVE_NOT_IN_LOBBY", "Сессия не в фазе lobby", http.StatusConflict)
	ErrLiveNotCompleted    = New("LIVE_NOT_COMPLETED", "Сессия ещё не завершена", http.StatusConflict)
	ErrLiveCodeExpired     = New("CODE_EXPIRED", "Код устарел — лобби закрыто или лайв уже начался", http.StatusGone)
	ErrWsTokenInvalid      = New("WS_TOKEN_INVALID", "ws_token недействителен или уже использован", http.StatusUnauthorized)
	ErrLiveKicked          = New("FORBIDDEN", "Участник кикнут из сессии", http.StatusForbidden)
	ErrLiveNotLibrary      = New("FORBIDDEN", "Квиз не является публичным библиотечным", http.StatusForbidden)

	// Вопросы
	ErrQuestionNotFound        = New("QUESTION_NOT_FOUND", "Вопрос не найден", http.StatusNotFound)
	ErrQuestionInvalidType     = New("VALIDATION_ERROR", "Неизвестный тип вопроса", http.StatusUnprocessableEntity)
	ErrQuestionOptionsRequired = New("VALIDATION_ERROR", "Необходимо указать варианты ответа", http.StatusUnprocessableEntity)
	ErrQuestionOneCorrect      = New("VALIDATION_ERROR", "Для single_choice должен быть ровно один правильный ответ", http.StatusUnprocessableEntity)
	ErrQuestionNoCorrect       = New("VALIDATION_ERROR", "Необходимо указать хотя бы один правильный ответ", http.StatusUnprocessableEntity)
	ErrQuestionMetadataInvalid = New("VALIDATION_ERROR", "Некорректная структура metadata для данного типа вопроса", http.StatusUnprocessableEntity)
)
