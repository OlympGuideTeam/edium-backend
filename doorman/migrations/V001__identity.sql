CREATE SCHEMA doorman;
SET search_path TO doorman;

CREATE TYPE identity_status AS ENUM (
    'active',
    'blocked',
    'deleted'
);

CREATE TABLE identity (
    id            TEXT PRIMARY KEY,
    phone         TEXT NOT NULL,
    status        identity_status NOT NULL,

    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT identity_phone_unique UNIQUE (phone)
);

CREATE OR REPLACE FUNCTION set_updated_at()
    RETURNS TRIGGER AS '
    BEGIN
        NEW.updated_at = now();
        RETURN NEW;
    END;
' LANGUAGE plpgsql;

CREATE TRIGGER trg_identity_updated_at
    BEFORE UPDATE ON identity
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
