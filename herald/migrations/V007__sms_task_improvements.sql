-- Идемпотентность: предотвращает дублирование SMS при повторе otp_sent-задачи
ALTER TABLE sms_task ADD COLUMN idempotency_key UUID;
CREATE UNIQUE INDEX idx_sms_task_idempotency ON sms_task (idempotency_key) WHERE idempotency_key IS NOT NULL;

-- Трассировка: сквозной traceparent от doorman.otp.sent до Android-шлюза
ALTER TABLE sms_task ADD COLUMN trace_ctx TEXT;

-- Клейм: защита от параллельного получения одной задачи несколькими Android-шлюзами
ALTER TABLE sms_task ADD COLUMN claimed_at TIMESTAMPTZ;

-- Retry при неудачном ack от Android-шлюза
ALTER TABLE sms_task ADD COLUMN retry_count INT NOT NULL DEFAULT 0;
ALTER TABLE sms_task ADD COLUMN max_retries INT NOT NULL DEFAULT 3;
