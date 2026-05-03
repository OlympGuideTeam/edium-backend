ALTER TABLE quiz_session ADD COLUMN live_host_user_id UUID;

CREATE INDEX idx_quiz_session_live_host_user ON quiz_session (live_host_user_id)
    WHERE live_host_user_id IS NOT NULL;
