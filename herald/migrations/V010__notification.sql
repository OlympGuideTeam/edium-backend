CREATE TABLE notification (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL,
    title      TEXT NOT NULL,
    body       TEXT NOT NULL,
    is_read    BOOLEAN NOT NULL DEFAULT false,
    data       JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notification_user_id ON notification (user_id, created_at DESC);
