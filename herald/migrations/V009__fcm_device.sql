CREATE TABLE fcm_device (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL,
    fcm_token  TEXT NOT NULL,
    platform   TEXT NOT NULL CHECK (platform IN ('ios', 'android')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_fcm_device_token UNIQUE (fcm_token)
);

CREATE INDEX idx_fcm_device_user_id ON fcm_device (user_id);
