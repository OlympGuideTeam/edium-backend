CREATE TABLE course (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    class_id      UUID         NOT NULL REFERENCES class(id) ON DELETE CASCADE,
    owner_id      UUID         NOT NULL REFERENCES "user"(id),
    title         TEXT         NOT NULL CHECK (title <> ''),
    module_count  INTEGER      NOT NULL DEFAULT 0,
    element_count INTEGER      NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX course_class_id_idx ON course(class_id);
CREATE INDEX course_owner_id_idx ON course(owner_id);

CREATE TRIGGER trg_course_updated_at
    BEFORE UPDATE ON course
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE course_module (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id     UUID         NOT NULL REFERENCES course(id) ON DELETE CASCADE,
    title         TEXT         NOT NULL CHECK (title <> ''),
    element_count INTEGER      NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX course_module_course_id_idx ON course_module(course_id);

CREATE TRIGGER trg_course_module_updated_at
    BEFORE UPDATE ON course_module
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE OR REPLACE FUNCTION update_course_module_count()
    RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE course SET module_count = module_count + 1 WHERE id = NEW.course_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE course SET module_count = module_count - 1 WHERE id = OLD.course_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_course_module_count
    AFTER INSERT OR DELETE ON course_module
    FOR EACH ROW
EXECUTE FUNCTION update_course_module_count();
