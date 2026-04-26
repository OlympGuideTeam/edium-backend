package apperr

var (
	ErrUnauthorized       = New("UNAUTHORIZED", "Требуется авторизация", 401)
	ErrUnauthorizedToken  = New("UNAUTHORIZED_TOKEN", "Невалидный токен", 401)
	ErrUnauthorizedClaims = New("UNAUTHORIZED_CLAIMS", "Невалидные claims в токене", 401)
	ErrUnauthorizedSub    = New("UNAUTHORIZED_SUB", "Невалидный subject в токене", 401)
	ErrNotFound           = New("NOT_FOUND", "Ресурс не найден", 404)
	ErrBadRequest         = New("BAD_REQUEST", "Неверный запрос", 400)
	ErrFileTooLarge       = New("FILE_TOO_LARGE", "Файл слишком большой", 400)
	ErrInvalidFileType    = New("INVALID_FILE_TYPE", "Невалидный тип файла", 400)
	ErrTooManyUploads     = New("TOO_MANY_UPLOADS", "Превышен лимит загрузок", 429)
	ErrInternalError      = New("INTERNAL_ERROR", "Внутренняя ошибка сервера", 500)
)
