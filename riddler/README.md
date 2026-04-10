## Интеграция с Charon (оценка свободных ответов)

Riddler публикует батч ответов студентов на один вопрос, Charon оценивает их через LLM и возвращает результат.

### Запрос: `charon.quiz.grade.requested`

```json
{
  "request_id": "uuid",
  "question": "Текст вопроса",
  "answers": [
    { "student_id": "a1b2c3d4", "text": "Ответ студента" }
  ]
}
```

- `request_id` — UUID, по нему Riddler сопоставляет ответ
- `question` — полный текст вопроса (тип `with_free_answer`)
- `answers` — все ответы на вопрос; если суммарный объём > 10 000 символов, Charon батчит сам

### Ответ: `charon.quiz.grade.completed`

```json
{
  "request_id": "uuid",
  "grades": [
    { "student_id": "a1b2c3d4", "score": 7, "comment": "Верно, но неполно" }
  ],
  "error": ""
}
```

- `score` — целое от 0 до 10; оценка **относительная**: если все ответили слабо, лучший получит не 10
- `comment` — краткий фидбек на русском
- `error` — непустая строка при ошибке (`RATE_LIMIT_EXCEEDED` и др.); в этом случае `grades` пустой

### Модель

Настраивается переменной `DEEPSEEK_GRADING_MODEL` в Charon (по умолчанию `deepseek-chat`).

---

### Поля квиза

- QuizID
- AuthorID
- Status (Draft, Deleted, Published)
- Name
- Description
- Сomplexity ?
- Time to question
- Display correct answers ?
- Publish to library?
- Question Number
- Questions

### Поля вопроса

- Position
- Text
- Image
- Type

### Типы вопросов

- single_choice (answers, correct_answer)
- multiple_choice (answers, correct_answers)
- with_given_answer (correct_answers)
- with_free_answer (llm_type)
- drag (correct_answers - with order)
- connection (correct_pairs)

### Взаимодействие с caesar
- если квиз проверился и он связан с курсом, то в caesar поедет оценка за квиз
