package apperr

type Error struct {
	Code        string
	Description string
	HTTPStatus  int
	Details     map[string]any
}

func (e *Error) Error() string {
	return e.Code
}

func New(code, desc string, status int) *Error {
	return &Error{
		Code:        code,
		Description: desc,
		HTTPStatus:  status,
	}
}

func (e *Error) WithDetails(d map[string]any) *Error {
	e.Details = d
	return e
}

func BadRequest(err error) *Error {
	return New(
		"VALIDATION_ERROR",
		"Ошибка валидации",
		400,
	).WithDetails(map[string]any{
		"error": err,
	})
}
