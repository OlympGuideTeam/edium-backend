ALTER TABLE course_draft ADD COLUMN title TEXT NOT NULL DEFAULT '';
ALTER TABLE course_draft ALTER COLUMN title DROP DEFAULT;

ALTER TABLE course_draft ADD CONSTRAINT uq_course_draft_quiz_course UNIQUE (quiz_template_id, course_id);
