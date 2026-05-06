ALTER TABLE quiz_session ADD COLUMN teacher_id UUID;

UPDATE quiz_session qs
SET teacher_id = COALESCE(qs.live_host_user_id, qt.author_id)
FROM quiz_template qt
WHERE qt.id = qs.quiz_template_id;

ALTER TABLE quiz_session ALTER COLUMN teacher_id SET NOT NULL;

DROP INDEX IF EXISTS idx_quiz_session_live_host_user;
ALTER TABLE quiz_session DROP COLUMN live_host_user_id;

CREATE INDEX idx_quiz_session_teacher ON quiz_session (teacher_id);
