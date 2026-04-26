CREATE TABLE generation_task (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id      UUID        NOT NULL UNIQUE,
    quiz_id     UUID        NOT NULL,
    text        TEXT        NOT NULL,
    trace_ctx   TEXT,
    status      VARCHAR(20) NOT NULL DEFAULT 'pending',
    -- pending → processing → done | failed
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at  TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    error       TEXT
);

CREATE TABLE generation_result (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id       UUID        NOT NULL UNIQUE,
    quiz_id      UUID        NOT NULL,
    payload      JSONB       NOT NULL,
    trace_ctx    TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ
);
