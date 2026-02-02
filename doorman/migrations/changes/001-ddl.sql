--liquibase formatted sql

--changeset doorman:1
CREATE SCHEMA IF NOT EXISTS doorman;
SET search_path TO doorman;

--rollback DROP SCHEMA doorman;

--changeset doorman:2
CREATE TYPE identity_status AS ENUM (
    'active',
    'blocked',
    'deleted'
);

--rollback DROP TYPE identity_status;

--changeset doorman:3
CREATE TABLE identity (
    id            TEXT PRIMARY KEY,
    phone         TEXT NOT NULL,
    status        identity_status NOT NULL,

    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT identity_phone_unique UNIQUE (phone)
);

--rollback DROP TABLE identity;

--changeset doorman:4
CREATE OR REPLACE FUNCTION set_updated_at()
    RETURNS TRIGGER AS '
    BEGIN
        NEW.updated_at = now();
        RETURN NEW;
    END;
' LANGUAGE plpgsql;

--rollback DROP FUNCTION set_updated_at();

--changeset doorman:5
CREATE TRIGGER trg_identity_updated_at
    BEFORE UPDATE ON identity
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

--rollback DROP TRIGGER trg_identity_updated_at ON identity;

--changeset doorman:6
CREATE TYPE task_status AS ENUM (
    'pending',
    'processing',
    'done',
    'failed'
);

--rollback DROP TYPE task_status;

--changeset doorman:7
CREATE TABLE task (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_type     TEXT NOT NULL,
    payload       JSONB NOT NULL,

    status        task_status NOT NULL DEFAULT 'pending',
    attempts      INT NOT NULL DEFAULT 0,
    max_attempts  INT NOT NULL DEFAULT 5,

    available_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_error    TEXT,

    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

--rollback DROP TABLE task;

--changeset doorman:8
CREATE INDEX idx_task_ready
    ON task (status, available_at)
    WHERE status = 'pending';

--rollback DROP INDEX idx_task_ready;

--changeset doorman:9
CREATE INDEX idx_async_tasks_type
    ON task (task_type);

--rollback DROP INDEX idx_async_tasks_type;

--changeset doorman:10
CREATE TRIGGER trg_task_updated_at
    BEFORE UPDATE ON task
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

--rollback DROP TRIGGER trg_task_updated_at ON task;
