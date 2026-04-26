CREATE TYPE attempt_status AS ENUM ('in_progress', 'grading', 'graded', 'completed');

CREATE TYPE final_source AS ENUM ('llm', 'teacher', 'auto');

CREATE TABLE attempt (
    id             UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id     UUID           NOT NULL REFERENCES quiz_session (id),
    user_id        UUID           NOT NULL,
    status         attempt_status NOT NULL DEFAULT 'in_progress',
    score          NUMERIC(10, 2),
    question_order JSONB,
    started_at     TIMESTAMPTZ    NOT NULL DEFAULT now(),
    finished_at    TIMESTAMPTZ,
    created_at     TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ    NOT NULL DEFAULT now()
);

CREATE INDEX idx_attempt_session ON attempt (session_id);
CREATE INDEX idx_attempt_user    ON attempt (user_id);

CREATE TRIGGER trg_attempt_updated_at
    BEFORE UPDATE ON attempt
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE answer_submission (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id     UUID         NOT NULL REFERENCES attempt (id) ON DELETE CASCADE,
    question_id    UUID         NOT NULL REFERENCES question (id),
    answer_data    JSONB        NOT NULL,
    final_score    NUMERIC(10, 2),
    final_source   final_source,
    final_feedback TEXT,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_answer_submission_attempt_question ON answer_submission (attempt_id, question_id);
CREATE INDEX idx_answer_submission_question ON answer_submission (question_id);

CREATE TRIGGER trg_answer_submission_updated_at
    BEFORE UPDATE ON answer_submission
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
