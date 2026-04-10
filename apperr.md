# Коды ошибок API

Все ошибочные ответы имеют единый формат:

```json
{
  "error": "ERROR_CODE",
  "description": "Текстовое описание",
  "details": {}
}
```

Константы хранятся в `<service>/internal/pkg/apperr/errors.go`.

---

## Doorman

| Константа | Код | HTTP | Описание |
|-----------|-----|------|----------|
| `ErrPhoneUnavailable` | `PHONE_UNAVAILABLE` | 403 | Пользователь с таким номером удален/заблокирован |
| `ErrOTPAlreadySent` | `OTP_ALREADY_SENT` | 429 | Одноразовый код уже отправлен |
| `ErrOTPDailyLimit` | `OTP_DAILY_LIMIT_EXCEEDED` | 429 | Превышен дневной лимит отправки кодов |
| `ErrOTPNotFoundOrExpired` | `OTP_NOT_FOUND_OR_EXPIRED` | 400 | Одноразовый код для данного номера не существует или истёк |
| `ErrOTPInvalid` | `OTP_INVALID` | 400 | Неверный код |
| `ErrOTPAttemptsExceeded` | `OTP_ATTEMPTS_EXCEEDED` | 429 | Слишком много попыток |
| `ErrRefreshTokenInvalid` | `REFRESH_TOKEN_INVALID` | 401 | Невалидный refresh токен |
| `ErrRefreshTokenExpired` | `REFRESH_TOKEN_EXPIRED` | 401 | Refresh токен истёк |
| `ErrRegTokenMissing` | `MISSING_REG_TOKEN` | 401 | Отсутствует токен регистрации |
| `ErrRegTokenInvalid` | `INVALID_REG_TOKEN` | 401 | Недействительный токен регистрации |

Дополнительно: `apperr.BadRequest(err)` — ошибка валидации входных данных (`VALIDATION_ERROR`, 400).

---

## Caesar

| Константа | Код | HTTP | Описание |
|-----------|-----|------|----------|
| **Авторизация** |
| `ErrUnauthorized` | `UNAUTHORIZED` | 401 | Требуется авторизация |
| `ErrUnauthorizedToken` | `UNAUTHORIZED` | 401 | Неверный или истёкший токен |
| `ErrUnauthorizedClaims` | `UNAUTHORIZED` | 401 | Неверные claims |
| `ErrUnauthorizedSub` | `UNAUTHORIZED` | 401 | Неверный формат sub в токене |
| **Общие** |
| `ErrBadRequest` | `BAD_REQUEST` | 400 | Некорректный запрос |
| `ErrBadID` | `BAD_REQUEST` | 400 | Некорректный идентификатор |
| `ErrInvalidRole` | `BAD_REQUEST` | 400 | Параметр role обязателен: teacher или student |
| **Пользователи** |
| `ErrUserNotFound` | `USER_NOT_FOUND` | 404 | Пользователь не найден |
| `ErrUserBlockedOrDeleted` | `FORBIDDEN` | 403 | Пользователь заблокирован или удалён |
| `ErrUserEmptyUpdateFields` | `BAD_REQUEST` | 400 | Необходимо указать хотя бы одно поле |
| **Классы** |
| `ErrClassNotFound` | `CLASS_NOT_FOUND` | 404 | Класс не найден |
| `ErrClassForbidden` | `FORBIDDEN` | 403 | Только владелец класса может выполнить это действие |
| `ErrClassRemoveOwner` | `FORBIDDEN` | 403 | Нельзя исключить владельца класса |
| `ErrClassEmptyTitle` | `VALIDATION_ERROR` | 422 | Название класса не может быть пустым |
| `ErrInvitationNotFound` | `INVITATION_NOT_FOUND` | 404 | Приглашение не найдено |
| `ErrAlreadyMember` | `ALREADY_MEMBER` | 409 | Пользователь уже является участником класса |
| `ErrMemberNotFound` | `MEMBER_NOT_FOUND` | 404 | Участник не найден в классе |
| **Курсы** |
| `ErrCourseNotFound` | `COURSE_NOT_FOUND` | 404 | Курс не найден |
| `ErrModuleNotFound` | `MODULE_NOT_FOUND` | 404 | Модуль не найден |
| `ErrCourseForbidden` | `FORBIDDEN` | 403 | Нет прав для выполнения этого действия |
| `ErrCourseEmptyTitle` | `VALIDATION_ERROR` | 422 | Название не может быть пустым |
| `ErrCourseNotMember` | `FORBIDDEN` | 403 | Вы не являетесь участником этого класса |
