# Riddler — схема данных

## База данных

```mermaid
erDiagram
    quiz_template {
        uuid    id                  PK
        uuid    author_id
        text    title
        text    description
        jsonb   default_settings
        bool    is_public
        bool    is_draft
        bool    need_evaluation
        int     question_count
        uuid    library_session_id  FK "nullable"
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
        ts            created_at
        ts            updated_at
    }

    answer_option {
        uuid id          PK
        uuid question_id FK
        text text
        bool is_correct
        ts   created_at
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
        ts             created_at
        ts             updated_at
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
        ts             created_at
        ts             updated_at
    }

    answer_submission {
        uuid         id             PK
        uuid         attempt_id     FK
        uuid         question_id    FK
        jsonb        answer_data
        numeric      final_score
        final_source final_source
        text         final_feedback
        ts           created_at
        ts           updated_at
    }

    quiz_template   ||--o{ question          : "содержит"
    question        ||--o{ answer_option     : "варианты"
    quiz_template   ||--o{ quiz_session      : "запускается как"
    quiz_template   }o--o| quiz_session      : "library_session_id"
    quiz_session    ||--o{ attempt           : "попытки"
    attempt         ||--o{ answer_submission : "ответы"
    question        ||--o{ answer_submission : "на вопрос"
```

## Перечисления

| Тип | Значения |
|-----|----------|
| `session_mode` | `live`, `test` |
| `session_status` | `not_started`, `waiting`, `active`, `running`, `finished` |
| `attempt_status` | `in_progress`, `grading`, `graded`, `completed` |
| `final_source` | `auto`, `llm`, `teacher` |
| `question_type` | `single_choice`, `multiple_choice`, `with_given_answer`, `with_free_answer`, `drag`, `connection` |

## NATS-события (Riddler публикует)

| Subject | Payload | Когда |
|---------|---------|-------|
| `riddler.quiz_template.attached` | `{quiz_template_id, module_id}` | POST /quizzes с `attach_to_module` |
| `riddler.course_session.created` | `{session_id, module_id}` | POST /sessions/test или /sessions/live |
| `riddler.attempt.created` | `{attempt_id, session_id, user_id}` | POST /sessions/:id/attempts |
| `riddler.attempt.scored` | `{attempt_id, session_id, user_id, total_score}` | POST /attempts/:id/finish (после автооценки) |

Caesar подписывается на все четыре события и создаёт/обновляет записи в своей БД.
