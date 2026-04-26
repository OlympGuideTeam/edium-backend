ALTER TABLE sms_task ADD COLUMN idempotency_key UUID;
CREATE UNIQUE INDEX idx_sms_task_idempotency ON sms_task (idempotency_key) WHERE idempotency_key IS NOT NULL;

ALTER TABLE sms_task ADD COLUMN trace_ctx TEXT;

ALTER TABLE sms_task ADD COLUMN claimed_at TIMESTAMPTZ;

ALTER TABLE sms_task ADD COLUMN retry_count INT NOT NULL DEFAULT 0;
ALTER TABLE sms_task ADD COLUMN max_retries INT NOT NULL DEFAULT 3;
