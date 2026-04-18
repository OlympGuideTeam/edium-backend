DELETE FROM course_item WHERE type = 'quiz_template';

CREATE TYPE course_item_type_new AS ENUM ('quiz');

ALTER TABLE course_item
    ALTER COLUMN type TYPE course_item_type_new
    USING type::text::course_item_type_new;

DROP TYPE course_item_type;

ALTER TYPE course_item_type_new RENAME TO course_item_type;

ALTER TABLE course_item DROP COLUMN settings;

CREATE TABLE course_draft (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_template_id UUID        NOT NULL,
    course_id        UUID        NOT NULL REFERENCES course(id) ON DELETE CASCADE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_course_draft_updated_at
    BEFORE UPDATE ON course_draft
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
