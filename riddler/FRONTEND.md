# Riddler — интеграция для фронтенда

Сервис квизов. Здесь создаются квизы, проводятся сессии и оцениваются ответы студентов.

Базовый префикс всех ручек: `/riddler/v1`
Авторизация: Bearer-токен в заголовке `Authorization`.

---

## Флоу 1 — Библиотека (самостоятельное прохождение)

Квиз опубликован в публичную библиотеку (`is_public=true`). Студент видит его в каталоге и проходит в своём темпе.

```
GET /quizzes/{id}?role=student          ← узнать название, лимиты, кол-во вопросов
        │
        │  в ответе: library_test_session_id (UUID сессии)
        ▼
POST /sessions/{session_id}/attempts    ← создать попытку, получить вопросы
        │
        │  в ответе: attempt_id + список вопросов (без правильных ответов)
        ▼
POST /attempts/{attempt_id}/answers     ← отправлять ответы по ходу теста
        │                                  (можно перезаписывать — upsert)
        │
        ▼
POST /attempts/{attempt_id}/finish      ← завершить попытку → автооценка
        │
        ▼
GET  /attempts/{attempt_id}/result      ← получить итог с баллами по каждому вопросу
```

---

## Флоу 2 — Курс: шаблон квиза в модуле (quiz_template)

Учитель создаёт квиз и сразу прикрепляет его к модулю курса. Квиз виден только учителям — студенты его не видят. Шаблон можно продолжать редактировать (добавлять/менять вопросы).

```
Учитель:

POST /quizzes
  body: { title, description, default_settings, attach_to_module: { module_id } }
        │
        ├── создаёт quiz_template (is_draft=true)
        └── публикует событие → Caesar создаёт course_item
                                  { type: quiz_template, object_id: quiz_template_id }

POST /quizzes/{id}/questions   ← добавить вопросы (квиз ещё черновик, редактируемый)

POST /quizzes/{id}/publish     ← опубликовать шаблон (is_draft → false)
```

На стороне Caesar:
- Учитель видит элемент модуля с типом `quiz_template`
- Может перейти в редактор квиза, изменить вопросы и настройки
- Студенты этот элемент не видят

---

## Флоу 3 — Курс: сессия квиза (quiz, назначение задания)

Учитель берёт опубликованный (`is_draft=false`) квиз и создаёт для курса сессию с конкретными настройками. Студенты видят задание и могут создать попытку.

```
Учитель:

POST /sessions/test
  body: {
    quiz_template_id,
    module_id,
    total_time_limit_sec,  ← опционально (fallback: default_settings → 60×кол-во вопросов)
    shuffle_questions,     ← опционально (fallback: default_settings)
    started_at,            ← опционально
    finished_at            ← дедлайн, опционально
  }
        │
        ├── создаёт quiz_session (mode=test, status=active)
        └── публикует событие → Caesar создаёт course_item
                                  { type: quiz, object_id: session_id }

Ответ: { session_id }
```

На стороне Caesar:
- Студент видит элемент модуля с типом `quiz`
- `object_id` = session_id — используется для создания попытки

```
Студент (через Caesar):

GET /courses/{id}   ← получить курс с модулями
                      в элементах типа quiz: object_id = session_id

POST /sessions/{object_id}/attempts   ← создать попытку (тот же флоу, что в библиотеке)
POST /attempts/{attempt_id}/answers
POST /attempts/{attempt_id}/finish
GET  /attempts/{attempt_id}/result
```

---

## Прогресс в курсе (события для Caesar)

При работе студента с курсовым квизом Riddler публикует события, Caesar обновляет прогресс:

| Событие | Когда | Что делает Caesar |
|---------|-------|------------------|
| `riddler.attempt.created` | Создана попытка | Создаёт запись прогресса (score=null) |
| `riddler.attempt.scored` | Завершена попытка, получена оценка | Обновляет score в прогрессе |

Caesar отображает `attempt_id` и `score` по каждому элементу в `GET /courses/{id}`.

---

## Важные детали прохождения

- **Порядок вопросов фиксирован** на момент создания попытки. Если в сессии включён `shuffle_questions`, порядок перемешивается один раз и сохраняется — при обновлении страницы вопросы не прыгают.
- **Таймер**: если в сессии задан `total_time_limit_sec`, попытка завершается автоматически по истечении времени. При следующем `POST /answers` придёт `ATTEMPT_EXPIRED`. Фронт должен сам считать таймер и вызывать `/finish` заранее.
- **Повторная отправка ответа** на уже отвеченный вопрос — это upsert, не ошибка. Последнее значение побеждает.
- `with_free_answer` **не оценивается автоматически** — оценит учитель или LLM позже. В результате по таким вопросам `final_score` будет `null` до ручной проверки.
- **Одна попытка** на курсовую сессию для каждого студента.

---

## Типы вопросов

### Как создаёт учитель

```
POST /quizzes/{id}/questions
```

```json
{
  "type": "...",
  "text": "Текст вопроса",
  "image_link": null,
  "max_score": 10,
  "answer_options": [...],
  "metadata": {...}
}
```

`answer_options` нужны только для `single_choice` и `multiple_choice`.
`metadata` используется для остальных типов — структура описана в таблице.

---

### Таблица типов

| Тип | Что создаёт учитель | Что получает студент | Что присылает студент | Оценка |
|-----|--------------------|-----------------------|-----------------------|--------|
| `single_choice` | `answer_options: [{text, is_correct}]` | `options: [{id, text}]` — без `is_correct` | `{"selected_option_id": "uuid"}` | Полный балл если верный вариант, иначе 0 |
| `multiple_choice` | `answer_options: [{text, is_correct}]` | `options: [{id, text}]` — без `is_correct` | `{"selected_option_ids": ["uuid1", "uuid2"]}` | Частичный: `max(0, (верных − неверных) / всего_верных) × max_score`, округление до 2 знаков |
| `with_given_answer` | `metadata: {"correct_answers": ["Paris", "paris"]}` | Нет `options`, нет `metadata` | `{"text": "Paris"}` | Полный балл при точном совпадении (регистр важен), иначе 0 |
| `with_free_answer` | Ничего лишнего | Нет `options`, нет `metadata` | `{"text": "Свободный ответ"}` | Не оценивается автоматически — `final_score: null` |
| `drag` | `metadata: {"correct_order": ["B", "A", "C"]}` | `metadata: {"items": ["A", "C", "B"]}` — перемешанный | `{"order": ["B", "A", "C"]}` | Полный балл если порядок совпадает точно, иначе 0 |
| `connection` | `metadata: {"left": ["A","B"], "right": ["1","2"], "correct_pairs": {"A":"1","B":"2"}}` | `metadata: {"left": ["A","B"], "right": ["2","1"]}` — правая колонка перемешана | `{"pairs": {"A": "1", "B": "2"}}` | Полный балл если все пары верны, иначе 0 |

### Частичная оценка multiple_choice — пример

Вопрос: 4 варианта, 2 правильных, `max_score = 10`.

| Выбрал студент | Формула | Балл |
|----------------|---------|------|
| Оба верных | (2−0)/2 × 10 | **10.00** |
| 1 верный, 0 неверных | (1−0)/2 × 10 | **5.00** |
| 1 верный, 1 неверный | (1−1)/2 × 10 | **0.00** |
| 0 верных, 1 неверный | max(0, (0−1)/2) × 10 | **0.00** |

---

## Интеграция с Charon (оценка свободных ответов)

Riddler публикует батч ответов на вопрос типа `with_free_answer` в очередь, Charon оценивает их через LLM и возвращает результат.

**Запрос** `charon.quiz.grade.requested`:
```json
{
  "request_id": "uuid",
  "question": "Текст вопроса",
  "answers": [
    { "student_id": "uuid", "text": "Ответ студента" }
  ]
}
```

**Ответ** `charon.quiz.grade.completed`:
```json
{
  "request_id": "uuid",
  "grades": [
    { "student_id": "uuid", "score": 7, "comment": "Верно, но неполно" }
  ],
  "error": ""
}
```

- `score` — от 0 до 10, **относительный**: если все ответили слабо, лучший может получить меньше 10
- `comment` — фидбек на русском, будет показан студенту
- При ошибке `error` непустой, `grades` пустой
