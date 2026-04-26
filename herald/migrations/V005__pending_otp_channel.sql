-- Добавляем канал (tg / vk) в pending_otp.
-- Один номер телефона может быть привязан к разным ботам одновременно,
-- поэтому первичный ключ становится составным: (phone, channel).
ALTER TABLE pending_otp ADD COLUMN channel TEXT NOT NULL DEFAULT 'tg';
ALTER TABLE pending_otp DROP CONSTRAINT pending_otp_pkey;
ALTER TABLE pending_otp ADD PRIMARY KEY (phone, channel);
ALTER TABLE pending_otp ALTER COLUMN channel DROP DEFAULT;
