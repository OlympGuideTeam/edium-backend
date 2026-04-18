# Caesar — схема данных

## База данных

```mermaid
erDiagram
    user {
        uuid        id         PK
        text        name
        text        surname
        text        phone      "UNIQUE"
        user_status status
        ts          created_at
        ts          updated_at
    }

    class {
        uuid  id            PK
        text  title
        uuid  owner_id      FK
        int   student_count "триггер"
        ts    created_at
        ts    updated_at
    }

    class_member {
        uuid              class_id FK
        uuid              user_id  FK
        class_member_role role
        ts                created_at
    }

    class_invitation {
        uuid              id       PK
        uuid              class_id FK
        class_member_role role
        ts                created_at
    }

    course {
        uuid  id            PK
        uuid  class_id      FK
        uuid  owner_id      FK
        text  title
        int   module_count  "триггер"
        int   element_count "триггер"
        ts    created_at
        ts    updated_at
    }

    course_module {
        uuid  id            PK
        uuid  course_id     FK
        text  title
        int   element_count "триггер"
        int   order_index
        ts    created_at
        ts    updated_at
    }

    course_item {
        uuid             id        PK
        uuid             module_id FK
        uuid             object_id "ссылка на объект в другом сервисе"
        course_item_type type
        jsonb            settings
        ts               created_at
        ts               updated_at
    }

    course_user_item_progress {
        uuid    id             PK
        uuid    course_item_id FK
        uuid    user_id        FK
        uuid    attempt_id     "nullable, ссылка на Riddler"
        numeric score
        ts      created_at
        ts      updated_at
    }

    user             ||--o{ class            : "владеет"
    user             ||--o{ class_member     : "участник"
    class            ||--o{ class_member     : "состав"
    class            ||--o{ class_invitation : "приглашения"
    class            ||--o{ course           : "курсы"
    user             ||--o{ course           : "автор"
    course           ||--o{ course_module    : "модули"
    course_module    ||--o{ course_item      : "элементы"
    course_item      ||--o{ course_user_item_progress : "прогресс"
    user             ||--o{ course_user_item_progress : "прогресс"
```

## Перечисления

| Тип | Значения |
|-----|----------|
| `user_status` | `active`, `blocked`, `deleted` |
| `class_member_role` | `teacher`, `student` |
| `course_item_type` | `quiz`, `quiz_template` |
