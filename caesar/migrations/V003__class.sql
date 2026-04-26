CREATE TYPE class_member_role AS ENUM ('teacher', 'student');

CREATE TABLE class (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    title          TEXT        NOT NULL,
    owner_id       UUID        NOT NULL REFERENCES "user"(id),
    student_count  INTEGER     NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_class_updated_at
    BEFORE UPDATE ON class
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE class_member (
    class_id   UUID             NOT NULL REFERENCES class(id) ON DELETE CASCADE,
    user_id    UUID             NOT NULL REFERENCES "user"(id),
    role       class_member_role NOT NULL,
    created_at TIMESTAMPTZ      NOT NULL DEFAULT now(),

    PRIMARY KEY (class_id, user_id)
);

CREATE OR REPLACE FUNCTION update_class_student_count()
    RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' AND NEW.role = 'student' THEN
        UPDATE class SET student_count = student_count + 1 WHERE id = NEW.class_id;
    ELSIF TG_OP = 'DELETE' AND OLD.role = 'student' THEN
        UPDATE class SET student_count = student_count - 1 WHERE id = OLD.class_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_class_member_student_count
    AFTER INSERT OR DELETE ON class_member
    FOR EACH ROW
EXECUTE FUNCTION update_class_student_count();
