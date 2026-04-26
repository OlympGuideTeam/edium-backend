CREATE TABLE pending_otp (
    correlation_id TEXT PRIMARY KEY,
    chat_id        BIGINT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at     TIMESTAMPTZ NOT NULL DEFAULT now() + INTERVAL '10 minutes'
);

CREATE INDEX idx_pending_otp_expires ON pending_otp (expires_at);
