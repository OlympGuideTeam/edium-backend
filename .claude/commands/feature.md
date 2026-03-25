Реализуй новую фичу в Go-сервисе Edium.

При реализации следуй структуре сервиса:

**1. Domain**
- `internal/domain/` — новые бизнес-объекты и статусы
- Чистый Go, никаких зависимостей от БД/HTTP

**2. Repository**
- `internal/repository/` — интерфейс + реализация с PostgreSQL
- SQL-запросы напрямую (sqlx или pgx), без ORM
- Новые таблицы → миграция в `migrations/V{N}__{desc}.sql`

**3. Service**
- `internal/service/` — бизнес-логика, оркестрация
- Зависит только от интерфейсов repository

**4. Handler + DTO**
- `internal/transport/dto/` — request/response структуры
- `internal/handler/` — HTTP-обработчик (валидация → сервис → ответ)
- HTTP-коды: 200, 400, 401, 422, 500

**5. Регистрация**
- `internal/app/` — подключить handler к роутеру

**6. Тесты**
- Unit-тест service с мок-repository
- Integration-тест для repository если нужна проверка SQL

Запусти после: `go test ./...` и `go vet ./...`
