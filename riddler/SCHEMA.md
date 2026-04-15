# Riddler — схема данных и события

## База данных

```mermaid
erDiagram
    quiz_template {
        uuid    id               PK
        uuid    author_id
        text    title
        text    description
        jsonb   default_settings
        bool    is_public
        bool    is_draft
        bool    need_evaluation
        int     question_count
        ts      created_at
        ts      updated_at
    }

    question {
        uuid          id               PK
        uuid          quiz_template_id FK
        question_type type
        text          text
        text          image_link
        int           order_index
        jsonb         metadata
        int           max_score
    }

    answer_option {
        uuid id          PK
        uuid question_id FK
        text text
        bool is_correct
    }

    quiz_session {
        uuid           id                      PK
        uuid           quiz_template_id        FK
        session_mode   mode
        session_status status
        int            total_time_limit_sec
        int            question_time_limit_sec
        bool           shuffle_questions
        jsonb          settings
        ts             started_at
        ts             finished_at
    }

    attempt {
        uuid           id             PK
        uuid           session_id     FK
        uuid           user_id
        attempt_status status
        numeric        score
        jsonb          question_order
        ts             started_at
        ts             finished_at
    }

    answer_submission {
        uuid         id             PK
        uuid         attempt_id     FK
        uuid         question_id    FK
        jsonb        answer_data
        numeric      final_score
        final_source final_source
        text         final_feedback
    }

    quiz_template   ||--o{ question         : "содержит"
    question        ||--o{ answer_option    : "варианты"
    quiz_template   ||--o{ quiz_session     : "запускается как"
    quiz_session    ||--o{ attempt          : "попытки"
    attempt         ||--o{ answer_submission : "ответы"
    question        ||--o{ answer_submission : "на вопрос"
```

## Типы сессий

| mode | Описание |
|------|----------|
| `test` | Студент проходит самостоятельно (библиотека или курс) |
| `live` | Учитель ведёт урок в реальном времени |

| status | Описание |
|--------|----------|
| `draft` | Создана, не активна |
| `active` | Студенты могут создавать попытки |
| `finished` | Закрыта, новые попытки невозможны |

## NATS-события (Riddler публикует)

| Subject | Payload | Когда |
|---------|---------|-------|
| `riddler.quiz_template.attached` | `{quiz_template_id, module_id}` | POST /quizzes с `attach_to_module` |
| `riddler.course_session.created` | `{session_id, module_id}` | POST /sessions (курсовая сессия) |
| `riddler.attempt.created` | `{attempt_id, session_id, user_id}` | POST /sessions/:id/attempts |
| `riddler.attempt.scored` | `{attempt_id, session_id, user_id, total_score}` | POST /attempts/:id/finish (после автооценки) |

Caesar подписывается на все четыре события и создаёт/обновляет записи в своей БД.

## Сессия библиотеки vs курсовая сессия

```
Библиотека                          Курс
──────────────────────────────────────────────────────
Одна session на всех студентов      Одна session на весь класс/группу
Создаётся автоматически             Создаётся учителем явно (POST /sessions)
  при PublishQuiz (is_public=true)
Настройки из default_settings       Настройки задаются вручную
  квиза                               (time_limit, shuffle, finished_at)
Без дедлайна                        С дедлайном (finished_at)
course_item — отсутствует           course_item type=quiz, object_id=session_id
```
