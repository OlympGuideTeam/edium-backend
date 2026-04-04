CREATE TABLE class_invitation (
    id         UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    class_id   UUID              NOT NULL REFERENCES class(id) ON DELETE CASCADE,
    role       class_member_role NOT NULL,
    created_at TIMESTAMPTZ       NOT NULL DEFAULT now(),

    UNIQUE (class_id, role)
);
