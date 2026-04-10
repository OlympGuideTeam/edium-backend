CREATE TYPE course_item_type AS ENUM ('quiz');

CREATE TABLE course_item (
    id          UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    module_id   UUID             NOT NULL REFERENCES course_module(id) ON DELETE CASCADE,
    ref_id      UUID             NOT NULL,
    type        course_item_type NOT NULL,
    order_index INTEGER          NOT NULL DEFAULT 0,
    settings    JSONB,
    created_at  TIMESTAMPTZ      NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ      NOT NULL DEFAULT now()
);

CREATE INDEX course_item_module_id_idx ON course_item(module_id);

CREATE TRIGGER trg_course_item_updated_at
    BEFORE UPDATE ON course_item
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE OR REPLACE FUNCTION update_course_element_count()
    RETURNS TRIGGER AS $$
DECLARE
    v_course_id UUID;
BEGIN
    IF TG_OP = 'INSERT' THEN
        SELECT course_id INTO v_course_id FROM course_module WHERE id = NEW.module_id;
        UPDATE course_module SET element_count = element_count + 1 WHERE id = NEW.module_id;
        UPDATE course SET element_count = element_count + 1 WHERE id = v_course_id;
    ELSIF TG_OP = 'DELETE' THEN
        SELECT course_id INTO v_course_id FROM course_module WHERE id = OLD.module_id;
        UPDATE course_module SET element_count = element_count - 1 WHERE id = OLD.module_id;
        UPDATE course SET element_count = element_count - 1 WHERE id = v_course_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_course_element_count
    AFTER INSERT OR DELETE ON course_item
    FOR EACH ROW
EXECUTE FUNCTION update_course_element_count();

CREATE TABLE course_user_item_progress (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    course_item_id UUID         NOT NULL REFERENCES course_item(id) ON DELETE CASCADE,
    user_id        UUID         NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    attempt_id     UUID,
    score          NUMERIC(5,2),
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (course_item_id, user_id)
);

CREATE INDEX course_user_item_progress_user_id_idx ON course_user_item_progress(user_id);

CREATE TRIGGER trg_course_user_item_progress_updated_at
    BEFORE UPDATE ON course_user_item_progress
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
