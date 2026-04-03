CREATE TYPE user_status AS ENUM (
    'active',
    'blocked',
    'deleted'
);

CREATE OR REPLACE FUNCTION set_updated_at()
    RETURNS TRIGGER AS $$
    BEGIN
        NEW.updated_at = now();
        RETURN NEW;
    END;
$$ LANGUAGE plpgsql;

CREATE TABLE "user" (
    id         UUID PRIMARY KEY,
    name       TEXT NOT NULL,
    surname    TEXT NOT NULL,
    phone      TEXT NOT NULL,
    status     user_status NOT NULL DEFAULT 'active',

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT user_phone_unique UNIQUE (phone)
);

CREATE TRIGGER trg_user_updated_at
    BEFORE UPDATE ON "user"
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
