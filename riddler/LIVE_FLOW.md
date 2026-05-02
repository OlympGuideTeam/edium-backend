# Riddler — Live-режим: полное описание флоу

Документ для разработчиков бэка и фронта. Описывает все шаги с обеих ролей:
какие ручки дёргать, в каком порядке, как реагировать на WebSocket-события.

---

## Содержание

1. [Понятия и термины](#1-понятия-и-термины)
2. [Типы live-сессий](#2-типы-live-сессий)
3. [Фазы сессии](#3-фазы-сессии)
4. [Хранение состояния в Redis](#4-хранение-состояния-в-redis)
5. [Защита от второго устройства](#5-защита-от-второго-устройства)
6. [Создание сессии (учитель)](#6-создание-сессии-учитель)
7. [Открытие лобби и подключение учителя](#7-открытие-лобби-и-подключение-учителя)
8. [Вход ученика в лобби](#8-вход-ученика-в-лобби)
9. [Лобби-экран: учитель видит, кто подключился](#9-лобби-экран-учитель-видит-кто-подключился)
10. [Лобби-экран: ученик видит других участников](#10-лобби-экран-ученик-видит-других-участников)
11. [Старт квиза](#11-старт-квиза)
12. [Вопрос активен: ученик отвечает](#12-вопрос-активен-ученик-отвечает)
13. [Вопрос активен: учитель мониторит](#13-вопрос-активен-учитель-мониторит)
14. [Закрытие вопроса (question_locked)](#14-закрытие-вопроса-question_locked)
15. [Переход к следующему вопросу](#15-переход-к-следующему-вопросу)
16. [Завершение квиза](#16-завершение-квиза)
17. [Итоговый экран ученика](#17-итоговый-экран-ученика)
18. [Итоговый экран учителя](#18-итоговый-экран-учителя)
19. [Реконнект и восстановление состояния](#19-реконнект-и-восстановление-состояния)
20. [Кик участника](#20-кик-участника)
21. [Ограничения и валидация](#21-ограничения-и-валидация)
22. [Коды ошибок](#22-коды-ошибок)
23. [Быстрые сводки](#23-быстрые-сводки)

---

## 1. Понятия и термины

| Термин | Значение |
|--------|----------|
| `session_id` | UUID live-сессии |
| `attempt_id` | UUID attempt'а участника. Единственный идентификатор участника во всех WS-событиях. |
| `user_id` | Профильный id зарегистрированного пользователя. Для library-анонимов — `null`. |
| `join_code` | 6-значный числовой код для входа в лобби. Активен только в фазе `lobby`. |
| `ws_token` | Одноразовый токен, выдаётся при `/live/join`. Сгорает при первом WS-handshake. |
| `source` | Тип сессии: `course` (привязана к модулю курса) или `library` (вне курса). |
| `phase` | Текущая фаза сессии. Подробнее — раздел 3. |
| `client_msg_id` | Необязательное поле в C→S сообщениях. Если передан — сервер вернёт `ack`/`error` с тем же значением, чтобы клиент мог понять, на какой именно запрос пришёл ответ. Удобно когда несколько сообщений отправляются подряд. Без него всё работает — ack просто не придёт. |

---

## 2. Типы live-сессий

### course-live

- Создаётся через `POST /sessions/live/course` (с `module_id`).
- Участвовать могут **только ученики класса**, к которому относится модуль.
- Авторизация: JWT обязателен.
- Имена участников передаются при `/live/join`; учитель видит их через WS-события.

### library-live

- Создаётся через `POST /sessions/live/library` (без курса).
- Участвовать могут **любые пользователи** по `join_code` — в том числе **без авторизации**.
- Неавторизованный пользователь вводит имя при входе в лобби; `user_id=null`, `name` сохраняется в `attempt`.
- Зарегистрированный пользователь может войти с JWT — тогда имя берётся из запроса (обязательно).
- Учитель видит участников только по WS-событиям `lobby.participant_joined`.

---

## 3. Фазы сессии

```
lobby → question_active ⇄ question_locked → completed
```

| Фаза | Когда наступает |
|------|----------------|
| `lobby` | Учитель вызвал `POST /sessions/{id}/live/start` |
| `question_active` | Учитель отправил `teacher.start_quiz` (первый) или `teacher.next_question` (последующие) |
| `question_locked` | Таймер истёк ИЛИ все активные участники ответили |
| `completed` | Учитель отправил `teacher.next_question` после последнего вопроса |

**Переход в `question_locked` происходит автоматически на сервере** — клиент ничего не инициирует.

> «Все активные участники» — только те, у кого WS-соединение живо (статус `connected` в Redis).
> Участники в состоянии `disconnected` (grace period истёк) не блокируют закрытие вопроса.

---

## 4. Хранение состояния в Redis

Активные live-сессии хранятся в Redis для быстрого доступа без обращения к PostgreSQL
на каждое WS-сообщение. В PostgreSQL пишется только финальное состояние.

### Структура ключей

```
live:{session_id}:phase                         STRING  текущая фаза (pending|lobby|...)
live:{session_id}:join_code                     STRING  6-значный код → TTL сбрасывается при start_quiz
live:{session_id}:current_question_idx          STRING  индекс текущего вопроса (0-based)
live:{session_id}:current_question_deadline     STRING  ISO-8601 timestamp дедлайна

live:{session_id}:participants                  HASH    attempt_id → JSON{user_id,name,status}
                                                        status: connected|disconnected|kicked

live:{session_id}:answers:{question_id}         HASH    attempt_id → JSON{answer_data,is_correct,score,time_ms}

live:{session_id}:stats:{question_id}           HASH    поля: answered, correct, avg_time_ms,
                                                        dist:{option_id} (для choice)

live:{session_id}:ws_tokens                     HASH    ws_token → attempt_id  (TTL на каждом поле)
live:{session_id}:active_ws                     HASH    attempt_id → conn_id   (активное соединение)
live:{session_id}:ws_grace:{attempt_id}         STRING  conn_id предыдущего коннекта, TTL=2с

live:code:{join_code}                           STRING  → session_id  (TTL до закрытия лобби)
```

### Жизненный цикл

- При создании сессии: инициализировать `phase=pending`, TTL всей группы ключей — 24 часа
  (защита от утечки памяти если лайв не был проведён).
- При переходе в `completed`: персистировать итоговые данные в PostgreSQL, затем удалить
  все ключи `live:{session_id}:*`.

---

## 5. Защита от второго устройства

Цель: один участник — одно активное WS-соединение в любой момент.

### ws_token как одноразовый пропуск

- `/live/start` и `/live/join` выдают `ws_token` — хранится в Redis
  с TTL 1 минута.
- При WS-handshake сервер **удаляет** токен из Redis — он сгорает сразу при использовании.
- Повторное подключение с тем же токеном → `401 WS_TOKEN_INVALID`.
- Чтобы зайти с другого устройства, нужно заново вызвать `POST /live/join`,
  который выдаст новый `ws_token` и аннулирует старый (если тот ещё не сгорел).

### Grace period 2 секунды при дисконнекте

При разрыве соединения сервер:
1. Записывает `live:{session_id}:ws_grace:{attempt_id} = <conn_id>` с TTL 2 секунды.
2. **Не** меняет статус участника сразу.

При новом WS-connect с `ws_token` для того же `attempt_id`:
- Если ключ `ws_grace:{attempt_id}` ещё жив → это реконнект того же клиента. Разрешить.
- Если ключ уже истёк и в `active_ws` записан другой `conn_id` → участник уже подключён
  с другого устройства → `4008 ALREADY_CONNECTED`.

При истечении grace period (TTL сработал, не было реконнекта):
- Установить статус участника `disconnected` в `live:{session_id}:participants`.
- Этот участник не учитывается в пороге «все ответили» до следующего переподключения.
- Послать `lobby.participant_left` всем (если фаза `lobby`).

---

## 6. Создание сессии (учитель)

### course-live

```
POST /riddler/v1/sessions/live/course
Authorization: Bearer <teacher_jwt>

{
  "quiz_template_id": "...",
  "module_id": "...",
  "question_time_limit_sec": 30   // опционально
}

→ 200 { "session_id": "uuid" }
```

### library-live

```
POST /riddler/v1/sessions/live/library
Authorization: Bearer <teacher_jwt>

{
  "quiz_template_id": "...",
  "question_time_limit_sec": 30   // опционально
}

→ 200 { "session_id": "uuid" }
```

**Валидация шаблона (оба варианта):**
- `need_evaluation` должен быть `false`.
- Ошибка: `422 LIVE_TEMPLATE_INVALID`.

---

## 7. Открытие лобби и подключение учителя

### Шаг 1 — учитель открывает лобби

```
POST /riddler/v1/sessions/{session_id}/live/start
Authorization: Bearer <teacher_jwt>

→ 200 {
  "ws_token": "...",
  "join_code": "471829"
}
```

Сервер инициализирует состояние в Redis (фаза `lobby`, join_code), генерирует ws_token (TTL 1 минута).

### Шаг 2 — учитель показывает код и подключается к WS

Учитель отображает `join_code` на экране. Одновременно подключается к WS:

```
WS wss://<host>/riddler/v1/sessions/{session_id}/live/ws?token=<ws_token>
```

Сервер сразу присылает `state.snapshot` с `phase=lobby`.

---

## 8. Вход ученика в лобби

### Вариант A — вход по коду

Ученик вводит 6-значный код вручную:

```
GET /riddler/v1/sessions/live/471829

→ 200 {
  "session_id": "uuid",
  "quiz_title": "Алгоритмы",
  "question_count": 12,
  "source": "library",
  "phase": "lobby",
  "is_anonymous_allowed": true,
  ...
}

// Если код истёк (лобби закрылось):
→ 410 CODE_EXPIRED
```

### Вариант B — вход по session_id (course, с экрана курса / главного экрана)

Клиент уже знает `session_id` — сразу переходит к join.

---

### Join — создать attempt и получить ws_token

```
POST /riddler/v1/sessions/{session_id}/live/join

// course — JWT обязателен, name не нужен
// library — name обязателен всегда (JWT опционально)
{ "name": "Вася" }

→ 200 {
  "attempt_id": "uuid",
  "ws_token": "..."
}
```

**После получения `ws_token`** клиент немедленно подключается к WS — токен живёт 1 минуту:

```
WS wss://<host>/riddler/v1/sessions/{session_id}/live/ws?token=<ws_token>
```

Сервер при handshake:
1. Проверяет `ws_token` в Redis, удаляет его (токен одноразовый).
2. Регистрирует новый `conn_id` в `active_ws`.
3. Шлёт `state.snapshot` с `phase=lobby` и текущим списком участников.
4. Рассылает всем `lobby.participant_joined`.

---

## 9. Лобби-экран: учитель видит, кто подключился

Для course-live клиент уже знает список учеников класса (получен через Caesar).
Клиент строит хэш-мапу `user_id → name` и отрисовывает весь класс как «ещё не подключился».

### WS-события в лобби (учитель)

```json
{ "type": "lobby.participant_joined", "data": {
    "attempt_id": "uuid",
    "user_id": "a1b2c3d4",
    "name": "Иван Петров"
}}
```

- **course**: находит `user_id` в хэш-мапе, помечает ученика как «подключился».
- **library**: добавляет нового участника по пришедшему `name` (хэш-мапы нет).

```json
{ "type": "lobby.participant_left", "data": { "attempt_id": "uuid" } }
```

Убрать из списка подключённых (ученик ушёл из лобби до старта).

---

## 10. Лобби-экран: ученик видит других участников

После подключения в `state.snapshot.lobby_participants` — текущий список участников в лобби.

Далее обновления по тем же событиям `lobby.participant_joined` / `lobby.participant_left`.
Ученик видит имена других и ждёт старта.

---

## 11. Старт квиза

Учитель нажимает «Начать»:

```json
{ "type": "teacher.start_quiz", "data": {} }
```

Сервер:
1. Деактивирует `join_code` (удаляет `live:code:{code}` из Redis).
2. Переводит фазу → `question_active`.
3. Шлёт всем `quiz.started`.
4. Шлёт всем `question.started` с первым вопросом.

**При получении `quiz.started`**: закрыть экран лобби.
**При получении `question.started`**: показать вопрос.

---

## 12. Вопрос активен: ученик отвечает

### Получение вопроса

```json
{ "type": "question.started", "data": {
    "question_index": 1,
    "question_total": 12,
    "question": {
        "id": "uuid",
        "type": "single_choice",
        "text": "Какая сложность алгоритма Дейкстры с бинарной кучей?",
        "options": [
            { "id": "uuid-a", "text": "O(V·E)" },
            { "id": "uuid-b", "text": "O((V+E)·log V)" },
            { "id": "uuid-c", "text": "O(V²)" },
            { "id": "uuid-d", "text": "O(E·log V)" }
        ],
        "max_score": 10
    },
    "time_limit_sec": 30,
    "deadline_at": "2024-01-01T12:00:30Z"
}}
```

UI ориентируется на `deadline_at` для синхронизации таймера с сервером
(компенсирует задержку доставки).

### Отправка ответа

```json
{ "type": "student.submit_answer", "data": {
    "question_id": "uuid",
    "answer_data": { "selected_option_id": "uuid-b" }
}}
```

Форматы `answer_data` по типу вопроса:

| Тип | Формат |
|-----|--------|
| `single_choice` | `{"selected_option_id": "uuid"}` |
| `multiple_choice` | `{"selected_option_ids": ["uuid1", "uuid2"]}` |
| `with_given_answer` | `{"text": "Paris"}` |
| `drag` | `{"order": ["B", "A", "C"]}` |
| `connection` | `{"pairs": {"A": "1", "B": "2"}}` |

После отправки: клиент показывает «вы ответили, ждите остальных».
Изменить ответ нельзя — повторный submit → `error: ALREADY_ANSWERED`.

Если все активные участники ответили раньше таймера — сервер автоматически
переходит в `question_locked`.

---

## 13. Вопрос активен: учитель мониторит

Учитель получает тот же `question.started`, но вопрос приходит с полными данными
(поле `is_correct` в опциях).

### Обновления в реальном времени

```json
{ "type": "participant.answered", "data": {
    "attempt_id": "uuid",
    "question_id": "uuid",
    "is_correct": true,
    "time_taken_ms": 8400
}}
```

Убрать ученика из списка «ещё думают», добавить в «ответили». Поля `is_correct` и `time_taken_ms` позволяют сразу обновить per-student индикаторы без ожидания `question.stats_tick`.

```json
{ "type": "question.stats_tick", "data": {
    "question_id": "uuid",
    "kind": "choice",
    "answered_count": 18,
    "connected_count": 24,
    "correct_count": 10,
    "incorrect_count": 8,
    "avg_time_ms": 21000,
    "distribution": [
        { "option_id": "uuid-a", "count": 3,  "is_correct": false },
        { "option_id": "uuid-b", "count": 10, "is_correct": true  },
        { "option_id": "uuid-c", "count": 4,  "is_correct": false },
        { "option_id": "uuid-d", "count": 1,  "is_correct": false }
    ]
}}
```

Поле `kind` — `"choice"` (single/multiple_choice) или `"binary"` (остальные типы).
При `kind="binary"` поле `distribution` отсутствует.
Приходит не чаще 1 раза в секунду — клиент просто перерисовывает текущие значения.

---

## 14. Закрытие вопроса (question_locked)

Событие присылается **всем** при наступлении одного из условий:
- Истёк таймер (`deadline_at`).
- Все **активные** (connected) участники отправили ответ.

### Ученик получает

```json
{ "type": "question.locked", "data": {
    "question_id": "uuid",
    "stats": {
        "question_id": "uuid",
        "kind": "choice",
        "answered_count": 24,
        "connected_count": 24,
        "correct_count": 14,
        "incorrect_count": 10,
        "avg_time_ms": 19500
    },
    "distribution": [
        { "option_id": "uuid-a", "count": 4,  "is_correct": false },
        { "option_id": "uuid-b", "count": 14, "is_correct": true  },
        { "option_id": "uuid-c", "count": 5,  "is_correct": false },
        { "option_id": "uuid-d", "count": 1,  "is_correct": false }
    ],
    "correct_answer": { "correct_option_id": "uuid-b" },
    "my_result": { "is_correct": true, "score": 10 }
}}
```

- `my_result=null` — если ученик не успел ответить до закрытия вопроса.
- `distribution` присутствует только при `kind="choice"`.
- `correct_answer` — структура зависит от типа вопроса (см. ниже).

### Учитель получает

То же, без поля `my_result`. Финальные `stats` и `distribution` вместо промежуточных тиков.

### Структура `correct_answer` по типам вопросов

| Тип | Поле | Пример |
|-----|------|--------|
| `single_choice` | `correct_option_id` | `{"correct_option_id": "uuid-b"}` |
| `multiple_choice` | `correct_option_ids` | `{"correct_option_ids": ["uuid-b", "uuid-d"]}` |
| `with_given_answer` | `correct_answers` | `{"correct_answers": ["Paris", "Париж"]}` |
| `drag` | `correct_order` | `{"correct_order": ["B", "A", "C"]}` |
| `connection` | `correct_pairs` | `{"correct_pairs": {"A": "1", "B": "2"}}` |

---

## 15. Переход к следующему вопросу

После `question.locked` оба экрана показывают статистику и правильный ответ.
Учитель управляет переходом:

```json
{ "type": "teacher.next_question", "data": {} }
```

Сервер:
- Остались вопросы → фаза `question_active`, шлёт всем `question.started`.
- Вопросы кончились → фаза `completed`, шлёт всем `quiz.completed`.

---

## 16. Завершение квиза

```json
{ "type": "quiz.completed", "data": {} }
```

Клиент (обе роли):
1. Закрывает WS.
2. Загружает итоги через REST.

---

## 17. Итоговый экран ученика

Без авторизации — идентификатор передаётся в query:

```
GET /riddler/v1/sessions/{session_id}/live/results/student?attempt_id=<uuid>

→ 200 {
  "my_position": 3,
  "total_participants": 24,
  "my_score": 118,
  "max_score": 120,
  "correct_count": 10,
  "questions_count": 12,
  "top": [
    { "position": 1, "attempt_id": "...", "name": "Мария Соколова", "score": 128, "is_me": false },
    { "position": 2, "attempt_id": "...", "name": "Артём Ким",       "score": 121, "is_me": false },
    { "position": 3, "attempt_id": "...", "name": "Вы",              "score": 118, "is_me": true  }
    // если позиция > 3 — добавляется строка is_me=true в конец
  ]
}
```

UI: мой ранг крупно + балл + топ-3 с подсветкой `is_me`.

---

## 18. Итоговый экран учителя

Для course-live — JWT обязателен (только автор квиза). Для library-live — без ограничений.

```
GET /riddler/v1/sessions/{session_id}/live/results/teacher
Authorization: Bearer <teacher_jwt>   // обязателен для course-live

→ 200 {
  "questions": [
    {
      "question_id": "...",
      "order_index": 1,
      "text": "Какая сложность Дейкстры?",
      "type": "single_choice",
      "correct_rate": 0.58,
      "stats": {
        "kind": "choice",
        "answered_count": 24,
        "correct_count": 14,
        "avg_time_ms": 21000,
        "distribution": [...]
      }
    },
    ...
  ],
  "leaderboard": [
    {
      "position": 1,
      "attempt_id": "...",
      "user_id": "a1b2c3d4",
      "name": "Мария Соколова",
      "score": 128,
      "max_score": 120,
      "correct_count": 11,
      "answers": [
        { "question_id": "...", "is_correct": true,  "score": 10 },
        { "question_id": "...", "is_correct": false, "score": 0  }
      ]
    },
    ...
  ]
}
```

`leaderboard` отсортирован по `score DESC`.

---

## 19. Реконнект и восстановление состояния

При любом (ре)коннекте сервер сразу шлёт **`state.snapshot`** — клиент восстанавливает
нужный экран без дополнительных запросов.

Снапшот всегда содержит поле `question_total` — общее число вопросов сессии.
Оно нужно для прогресс-бара «Вопрос X / Y» при реконнекте в середине квиза.

| Фаза в снапшоте | Экран |
|-----------------|-------|
| `lobby` | Лобби (обе роли) |
| `question_active` | Активный вопрос. `question_total` в снапшоте. Таймер: `remaining = deadline_at - now()`. Если ученик уже ответил — показать состояние ожидания. |
| `question_locked` | Экран статистики. Поле `last_question_locked` в снапшоте содержит `correct_answer`, `stats`, `distribution`, `my_result`. |
| `completed` | Переход на экран итогов. |

**Механизм реконнекта (ученик):**
1. Соединение разорвалось.
2. Клиент вызывает `POST /live/join` заново → получает новый `ws_token`.
3. Подключается к WS → получает `state.snapshot`.

Grace period 2 секунды гарантирует, что кратковременный обрыв (например, переход
между вышками) не помечает участника как `disconnected` и не срабатывает порог
«все ответили» раньше времени.

---

## 20. Кик участника

Учитель:

```json
{ "type": "teacher.kick_participant", "data": { "attempt_id": "uuid" } }
```

Сервер:
1. Шлёт `participant.kicked {attempt_id}` **всем** — убрать из списка.
2. Шлёт `you_were_kicked` **кикнутому**, закрывает его WS с кодом `4003`.
3. Помечает `attempt` как `kicked` в Redis и PostgreSQL.
4. Блокирует повторный `POST /live/join` для этого участника (`403`).

---

## 21. Ограничения и валидация

| Правило | Ошибка |
|---------|--------|
| Нельзя создать live-сессию если `need_evaluation=true` | `422 LIVE_TEMPLATE_INVALID` |
| Нельзя создать live-сессию если в шаблоне есть `with_free_answer` | `422 LIVE_TEMPLATE_INVALID` |
| Вход в лобби только если `phase=lobby` | `409 LIVE_NOT_IN_LOBBY` |
| Вход в course-live только участнику класса | `403` |
| `name` обязателен для library без JWT | `422` |
| `ws_token` одноразовый — повторное использование запрещено | `401 WS_TOKEN_INVALID` |
| Новый WS-connect пока участник ещё connected (и grace period не истёк) | WS close `4008 ALREADY_CONNECTED` |
| Ответ принимается только в `question_active` | WS `error: ANSWER_TOO_LATE` |
| Повторный ответ на тот же вопрос запрещён | WS `error: ALREADY_ANSWERED` |
| `teacher.next_question` только в `question_locked` | WS `error: NOT_IN_QUESTION_LOCKED` |
| `teacher.start_quiz` только в `lobby` | WS `error: NOT_IN_LOBBY` |
| Кикнутый не может переподключиться | `403` на `/live/join` |
| Второй WS-коннект учителя вытесняет первый | WS close `4009 REPLACED` на старом соединении |
| `join_code` деактивируется при переходе `lobby → question_active` | `410 CODE_EXPIRED` |

---

## 22. Коды ошибок

### HTTP

| Статус | Error | Значение |
|--------|-------|----------|
| 422 | `LIVE_TEMPLATE_INVALID` | Шаблон содержит `with_free_answer` или `need_evaluation=true` |
| 409 | `LIVE_NOT_IN_LOBBY` | Попытка join вне фазы `lobby` |
| 409 | `LIVE_NOT_COMPLETED` | GET results до завершения |
| 410 | `CODE_EXPIRED` | `join_code` деактивирован |
| 401 | `WS_TOKEN_INVALID` | `ws_token` уже использован или истёк |

### WebSocket close codes

| Код | Значение |
|-----|----------|
| `4003` | Участник кикнут учителем |
| `4008` | Попытка подключиться пока соединение ещё активно (другое устройство) |
| `4009` | Соединение учителя вытеснено новым коннектом |

### WebSocket события `error`

| code | Значение |
|------|----------|
| `ANSWER_TOO_LATE` | Ответ после закрытия вопроса |
| `ALREADY_ANSWERED` | Повторный ответ на тот же вопрос |
| `INVALID_ANSWER_DATA` | Некорректный формат `answer_data` |
| `NOT_IN_LOBBY` | `teacher.start_quiz` вне фазы `lobby` |
| `NOT_IN_QUESTION_LOCKED` | `teacher.next_question` вне фазы `question_locked` |

---

## 23. Быстрые сводки

### REST-ручки

| Ручка | Кто | Когда |
|-------|-----|-------|
| `POST /sessions/live/course` | Учитель | Создание course-live |
| `POST /sessions/live/library` | Учитель | Создание library-live |
| `POST /sessions/{id}/live/start` | Учитель | Открыть лобби → `{ws_token, join_code}` |
| `GET /sessions/live/{code}` | Ученик | Резолв кода → превью сессии |
| `POST /sessions/{id}/live/join` | Ученик | Войти в лобби → `{attempt_id, ws_token}` |
| `GET /sessions/{id}/live/ws` | Оба | WebSocket upgrade |
| `GET /sessions/{id}/live/results/student` | Ученик | Итоги после `completed` |
| `GET /sessions/{id}/live/results/teacher` | Учитель | Итоги после `completed` |

### WS C→S (клиент → сервер)

| Сообщение | Кто | Когда |
|-----------|-----|-------|
| `teacher.start_quiz` | Учитель | Стартовать квиз (`lobby→question_active`) |
| `teacher.next_question` | Учитель | Следующий вопрос или завершение |
| `teacher.kick_participant` | Учитель | Кикнуть ученика |
| `student.submit_answer` | Ученик | Отправить ответ на текущий вопрос |

### WS S→C (сервер → клиент)

| Событие | Кому | Когда |
|---------|------|-------|
| `state.snapshot` | Оба | При подключении / реконнекте |
| `lobby.participant_joined` | Оба | Новый участник в лобби |
| `lobby.participant_left` | Оба | Участник ушёл из лобби |
| `quiz.started` | Оба | Учитель нажал «Начать» |
| `question.started` | Оба | Новый вопрос активен |
| `participant.answered` | Учитель | Ученик ответил |
| `question.stats_tick` | Учитель | Промежуточная статистика (≤1/с) |
| `question.locked` | Оба | Вопрос закрыт, показать правильный ответ |
| `quiz.completed` | Оба | Все вопросы пройдены |
| `participant.kicked` | Оба | Участник кикнут |
| `you_were_kicked` | Кикнутый | Вы кикнуты |
| `ack` | Отправитель | Подтверждение C→S (если был `client_msg_id`) |
| `error` | Отправитель | Ошибка операции |
