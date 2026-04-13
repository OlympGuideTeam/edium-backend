CREATE TYPE question_type AS ENUM (
    'single_choice',
    'multiple_choice',
    'with_given_answer',
    'with_free_answer',
    'drag',
    'connection'
);

-- Шаблон квиза (авторский контент учителя)

CREATE TABLE quiz_template (
    id               UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id        UUID    NOT NULL,
    title            TEXT    NOT NULL,
    description      TEXT,
    default_settings JSONB   NOT NULL DEFAULT '{}',
    is_public        BOOLEAN NOT NULL DEFAULT false,
    is_draft         BOOLEAN NOT NULL DEFAULT true,
    need_evaluation  BOOLEAN NOT NULL DEFAULT false,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_quiz_template_author ON quiz_template (author_id);

CREATE TRIGGER trg_quiz_template_updated_at
    BEFORE UPDATE ON quiz_template
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- Вопросы квиза

CREATE TABLE question (
    id               UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_template_id UUID          NOT NULL REFERENCES quiz_template (id) ON DELETE CASCADE,
    type             question_type NOT NULL,
    text             TEXT          NOT NULL,
    image_link       TEXT,
    order_index      INT           NOT NULL,
    metadata         JSONB,
    max_score        INT           NOT NULL DEFAULT 10,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX idx_question_quiz_template ON question (quiz_template_id, order_index);

CREATE TRIGGER trg_question_updated_at
    BEFORE UPDATE ON question
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- Варианты ответов на вопрос

CREATE TABLE answer_option (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id UUID        NOT NULL REFERENCES question (id) ON DELETE CASCADE,
    text        TEXT        NOT NULL,
    is_correct  BOOLEAN     NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_answer_option_question ON answer_option (question_id);
