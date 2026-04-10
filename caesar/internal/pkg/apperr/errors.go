package apperr

var (
	ErrUnauthorized   = New("UNAUTHORIZED", "Требуется авторизация", 401)
	ErrBadRequest     = New("BAD_REQUEST", "Некорректный запрос", 400)
	ErrBadID          = New("BAD_REQUEST", "Некорректный идентификатор", 400)
	ErrInvalidRole    = New("BAD_REQUEST", "Параметр role обязателен: teacher или student", 400)
	ErrMemberNotFound = New("MEMBER_NOT_FOUND", "Участник не найден в классе", 404)
)
