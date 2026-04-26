CREATE TABLE image (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_name  TEXT NOT NULL,
    mime_type  TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    width      INT,
    height     INT,
    s3_key     TEXT NOT NULL UNIQUE,
    bucket_name TEXT NOT NULL DEFAULT 'edium-images',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_image_deleted_at ON image(deleted_at);
CREATE INDEX idx_image_s3_key ON image(s3_key);
