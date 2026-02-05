CREATE TYPE task_status AS ENUM (
    'pending',
    'processing',
    'done',
    'failed'
);

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

CREATE INDEX idx_task_ready
    ON task (status, available_at)
    WHERE status = 'pending';

CREATE INDEX idx_async_tasks_type
    ON task (task_type);

CREATE TRIGGER trg_task_updated_at
    BEFORE UPDATE ON task
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
