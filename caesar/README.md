### Документация API

```bash
npx @redocly/cli build-docs api/openapi.yaml -o api/index.html
```

Открой `api/index.html` в браузере.

### Сущности

- User
- Class
- Membership
- Invitation
- Course
- Element

### Ручки

Личный кабинет
- `GET`    `/caesar/v1/users/me`
- `PATCH`  `/caesar/v1/users/me`
- `DELETE` `/caesar/v1/users/me`

Классы
- `POST` `/caesar/v1/classes`
- `GET` `/caesar/v1/classes/{classId}`
- `PATCH` `/caesar/v1/classes/{classId}`
- `DELETE` `/caesar/v1/classes/{classId}`
- `GET` `/caesar/v1/classes/me` - классы, где я владелец, ученик, учитель

Приглашения
- `GET`   `/caesar/v1/classes/{classId}/invitations`
- `POST`   `/caesar/v1/invitations/{invitationId}/accept`

Участия
- `GET` `/caesar/v1/classes/{classId}/members`
- `DELETE`  `/caesar/v1/classes/{classId}/members/{userId}`

Курсы
- `POST` `/caesar/v1/courses`
- `GET` `/caesar/v1/classes/{classId}/courses/`
- `GET` `/caesar/v1/courses/{courseId}`
- `PATCH` `/caesar/v1/courses/{courseId}`
- `DELETE` `/caesar/v1/courses/{courseId}`
- `GET` `/caesar/v1/courses/me` - курсы, где я учитель и участник класса

Элементы контроля
- `POST` `/caesar/v1/courses/{courseId}/elements`
- `GET` `/caesar/v1/courses/{courseId}/elements`
- `DELETE`  `/caesar/v1/elements/{elementId}`
- `GET`     `/caesar/v1/elements/{elementId}`

Ведомость
- `GET` `/caesar/v1/courses/{courseId}/sheet`
