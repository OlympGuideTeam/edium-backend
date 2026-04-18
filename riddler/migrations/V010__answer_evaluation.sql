CREATE TYPE evaluation_status AS ENUM ('pending', 'completed', 'failed');

CREATE TABLE answer_evaluation (
    id            UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id UUID              NOT NULL REFERENCES answer_submission(id) ON DELETE CASCADE,
    status        evaluation_status NOT NULL DEFAULT 'completed',
    score         NUMERIC(10, 2),
    source        final_source      NOT NULL,
    feedback      TEXT,
    created_at    TIMESTAMPTZ       NOT NULL DEFAULT now()
);

CREATE INDEX idx_answer_evaluation_submission ON answer_evaluation (submission_id);
CREATE INDEX idx_answer_evaluation_pending_llm ON answer_evaluation (submission_id, source, status);

ALTER TABLE quiz_session ADD COLUMN grading_sent_at TIMESTAMPTZ;
CREATE INDEX idx_quiz_session_grading ON quiz_session (status) WHERE grading_sent_at IS NULL;

CREATE INDEX idx_attempt_session_status ON attempt (session_id, status);
CREATE INDEX idx_attempt_status ON attempt (status);
