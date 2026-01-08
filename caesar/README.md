### Сущности

- User
- Class
- Membership
- Invitation
- Course
- Element

### Ручки

Другие пользователи
- `GET`    `/caesar/v1/users`
- `GET`    `/caesar/v1/users/{userId}`

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
- `POST`   `/caesar/v1/classes/{classId}/invitations`
- `GET`    `/caesar/v1/invitations/me`
- `POST`   `/caesar/v1/invitations/{invitationId}/accept`
- `POST`   `/caesar/v1/invitations/{invitationId}/reject`

Участия
- `GET` `/caesar/v1/classes/{classId}/members`
- `DELETE`  `/caesar/v1/classes/{classId}/members/{userId}`

Курсы 
- `POST` `/caesar/v1/courses`
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
