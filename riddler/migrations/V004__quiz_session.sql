CREATE TYPE session_mode AS ENUM ('live', 'test');

CREATE TYPE session_status AS ENUM ('draft', 'waiting', 'active', 'finished');

CREATE TABLE quiz_session (
    id                      UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_template_id        UUID           NOT NULL REFERENCES quiz_template (id),
    mode                    session_mode   NOT NULL,
    course_item_id          UUID,
    total_time_limit_sec    INT,
    question_time_limit_sec INT,
    shuffle_questions       BOOLEAN,
    status                  session_status NOT NULL DEFAULT 'draft',
    settings                JSONB,
    started_at              TIMESTAMPTZ    NOT NULL DEFAULT now(),
    finished_at             TIMESTAMPTZ,
    created_at              TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ    NOT NULL DEFAULT now()
);

CREATE INDEX idx_quiz_session_quiz_template ON quiz_session (quiz_template_id);

CREATE TRIGGER trg_quiz_session_updated_at
    BEFORE UPDATE ON quiz_session
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
