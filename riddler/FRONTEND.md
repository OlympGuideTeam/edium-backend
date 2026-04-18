# Riddler — интеграция для фронтенда

Сервис квизов. Здесь создаются квизы, проводятся сессии и оцениваются ответы студентов.

Базовый префикс всех ручек: `/riddler/v1`
Авторизация: Bearer-токен в заголовке `Authorization`.

---

## Флоу 1 — Библиотека (самостоятельное прохождение)

Квиз опубликован в публичную библиотеку (`is_public=true`). Студент видит его в каталоге и проходит в своём темпе. При публикации автоматически создаётся одна библиотечная тест-сессия на всех — её id возвращается в `library_test_session_id`.

```
GET /quizzes/{id}?role=student          ← название, лимиты, кол-во вопросов
        │
        │  в ответе: library_test_session_id
        ▼
POST /sessions/{session_id}/attempts    ← создать попытку, получить вопросы
        │
        │  в ответе: attempt_id + список вопросов (без правильных ответов)
        ▼
POST /attempts/{attempt_id}/answers     ← отправлять ответы (upsert)
        ▼
POST /attempts/{attempt_id}/finish      ← завершить → автооценка
        ▼
GET  /attempts/{attempt_id}/result      ← итог с баллами по каждому вопросу
```

---

## Флоу 2 — Курс: шаблон квиза в модуле

Учитель создаёт квиз и сразу прикрепляет к модулю. Шаблон виден только учителям, редактируется до публикации.

```
Учитель:

POST /quizzes
  body: { title, description, default_settings, attach_to_course: { course_id } }
        │
        ├── создаёт quiz_template (source=course)
        └── событие → Caesar добавляет шаблон в курс

POST /quizzes/{id}/questions         ← добавить вопросы вручную
POST /quizzes/{id}/generate          ← сгенерировать вопросы по тексту (async)
  body: { "text": "параграф или статья" }
  → возвращает { job_id }; вопросы добавятся в квиз, когда Sphinx ответит

POST /quizzes/{id}/publish           ← опубликовать (is_public → true)
```

Студенты элемент типа `quiz_template` не видят.

---

## Флоу 3 — Курс: тест-сессия (задание с дедлайном)

Учитель берёт опубликованный квиз и назначает задание с окном доступа. Студент видит элемент типа `quiz` и проходит в своём темпе.

```
Учитель:

POST /sessions/test
  body: {
    quiz_template_id,
    module_id,
    total_time_limit_sec,  ← опционально (fallback: default_settings → 60×вопросов)
    shuffle_questions,     ← опционально (fallback: default_settings)
    started_at,            ← начало окна, опционально
    finished_at            ← дедлайн, опционально
  }
        │
        ├── создаёт quiz_session (mode=test, status=active)
        └── событие → Caesar создаёт course_item { type: quiz, object_id: session_id }

Ответ: { session_id }
```

```
Студент:

GET /courses/{id}   ← из Caesar; в элементах quiz: object_id = session_id

POST /sessions/{object_id}/attempts   ← создать попытку
POST /attempts/{attempt_id}/answers
POST /attempts/{attempt_id}/finish
GET  /attempts/{attempt_id}/result
```

Статус тест-сессии вычисляется из `started_at`/`finished_at`:

| Статус | Условие | Что показать |
|--------|---------|--------------|
| `not_started` | `now < started_at` | «Откроется [дата]» |
| `active` | в окне доступа | Можно начать |
| `finished` | `now > finished_at` или закрыта учителем | Недоступно |

Коды ошибок при `POST /sessions/:id/attempts`:

| Код | Причина |
|-----|---------|
| `SESSION_NOT_STARTED` | `now < started_at` |
| `SESSION_DEADLINE_PASSED` | `now > finished_at` |
| `SESSION_NOT_ACTIVE` | Учитель явно закрыл сессию |

---

## Флоу 4 — Курс: лайв-сессия (урок в реальном времени)

Учитель ведёт урок — все на одном вопросе, переходы управляет учитель. Все переходы статусов — только учитель.

```
Учитель:

POST /sessions/live
  body: {
    quiz_template_id,
    module_id,
    question_time_limit_sec  ← опционально (fallback: default_settings → 30с)
  }
  → создаёт quiz_session (mode=live, status=not_started)

[открывает лобби]  → status: not_started → waiting
[запускает квиз]   → status: waiting → running
[завершает]        → status: running → finished
```

```
Студент:

POST /sessions/{session_id}/attempts   ← только когда status=waiting (лобби)
```

| Статус | Что показать студенту |
|--------|-----------------------|
| `not_started` | Сессия запланирована, лобби ещё не открыто |
| `waiting` | Лобби открыто — можно войти |
| `running` | Квиз идёт, присоединиться нельзя |
| `finished` | Завершена |

---

## Важные детали прохождения

- **Порядок вопросов фиксирован** на момент создания попытки. Если включён `shuffle_questions` — перемешивается один раз и сохраняется.
- **Таймер**: если задан `total_time_limit_sec`, попытка завершается автоматически. При следующем `POST /answers` придёт `ATTEMPT_EXPIRED`. Фронт должен сам считать таймер и вызывать `/finish` заранее.
- **Повторная отправка ответа** — upsert, не ошибка. Последнее значение побеждает.
- `with_free_answer` **не оценивается автоматически** — `final_score: null` до ручной или LLM-проверки.
- **Одна попытка** на курсовую сессию для каждого студента.
- **Статусы попытки для студента**: после `/finish` с `need_evaluation=true` попытка переходит в `grading`, затем в `graded` (LLM завершил), затем в `completed` (учитель завершил проверку). `GET /attempts/{id}/result` возвращает статус — фронт должен опрашивать до `completed`.

---

## Типы вопросов

```
POST /quizzes/{id}/questions
body: { type, text, image_link, max_score, answer_options, metadata }
```

`answer_options` — только для `single_choice` и `multiple_choice`.
`metadata` — для остальных типов.

| Тип | Что создаёт учитель | Что получает студент | Что присылает студент | Оценка |
|-----|--------------------|-----------------------|-----------------------|--------|
| `single_choice` | `answer_options: [{text, is_correct}]` | `options: [{id, text}]` | `{"selected_option_id": "uuid"}` | Полный балл если верный, иначе 0 |
| `multiple_choice` | `answer_options: [{text, is_correct}]` | `options: [{id, text}]` | `{"selected_option_ids": ["uuid1","uuid2"]}` | `max(0, (верных − неверных) / всего_верных) × max_score` |
| `with_given_answer` | `metadata: {"correct_answers": ["Paris","paris"]}` | нет options/metadata | `{"text": "Paris"}` | Полный балл при точном совпадении, иначе 0 |
| `with_free_answer` | — | — | `{"text": "Свободный ответ"}` | Не оценивается автоматически |
| `drag` | `metadata: {"correct_order": ["B","A","C"]}` | `metadata: {"items": ["A","C","B"]}` (перемешан) | `{"order": ["B","A","C"]}` | Полный балл если порядок точный, иначе 0 |
| `connection` | `metadata: {"left":["A","B"],"right":["1","2"],"correct_pairs":{"A":"1","B":"2"}}` | `metadata: {"left":["A","B"],"right":["2","1"]}` (правая перемешана) | `{"pairs":{"A":"1","B":"2"}}` | Полный балл если все пары верны, иначе 0 |

### Частичная оценка multiple_choice

Вопрос: 4 варианта, 2 правильных, `max_score = 10`.

| Выбрал студент | Формула | Балл |
|----------------|---------|------|
| Оба верных | (2−0)/2 × 10 | **10.00** |
| 1 верный, 0 неверных | (1−0)/2 × 10 | **5.00** |
| 1 верный, 1 неверный | (1−1)/2 × 10 | **0.00** |
| 0 верных, 1 неверный | max(0, (0−1)/2) × 10 | **0.00** |

---

---

## Флоу 5 — Проверка учителем (need_evaluation=true)

Применяется для тест-сессий, где квиз создан с `need_evaluation=true`. После сдачи попыток учитель видит статус `graded` (LLM уже выставил предварительные оценки) и может откорректировать любой ответ перед финальным завершением.

```
Учитель:

GET  /sessions/{session_id}/attempts
        │  в ответе: [{attempt_id, user_id, status, score}, ...]
        │  фильтровать по status=graded (ждут проверки)
        ▼
GET  /attempts/{attempt_id}/review
        │  в ответе: все ответы с question_text, answer_data,
        │            final_score, final_source (auto/llm), final_feedback
        ▼
POST /attempts/{attempt_id}/submissions/{submission_id}/grade   ← опционально
  body: { "score": 8, "feedback": "Неполный ответ" }
        │  можно вызывать несколько раз для разных submission_id
        │  перезаписывает оценку auto/llm, final_source становится teacher
        ▼
POST /attempts/{attempt_id}/complete
        │  пересчитывает итоговый балл, status: graded → completed
        └── событие riddler.attempt.scored → Caesar обновляет оценку
```

**Студент** после завершения проверки может открыть `GET /attempts/{id}/review` и увидеть финальные оценки с комментариями учителя.

| Статус попытки | Что показать студенту |
|----------------|----------------------|
| `in_progress` | Тест идёт |
| `grading` | На проверке |
| `graded` | Предварительная оценка (может измениться) |
| `completed` | Финальная оценка |

---

## Интеграция с Charon (оценка свободных ответов)

Riddler публикует батч на `riddler.quiz.grade.requested`, Charon оценивает через LLM и возвращает результат на `charon.quiz.grade.completed`.

**Запрос:**
```json
{
  "request_id": "uuid",
  "question": "Текст вопроса",
  "answers": [{ "student_id": "uuid", "text": "Ответ студента" }]
}
```

**Ответ:**
```json
{
  "request_id": "uuid",
  "grades": [{ "student_id": "uuid", "score": 7, "comment": "Верно, но неполно" }],
  "error": ""
}
```

- `score` — от 0 до 10, относительный
- `comment` — фидбек на русском, показывается студенту
- При ошибке `error` непустой, `grades` пустой
