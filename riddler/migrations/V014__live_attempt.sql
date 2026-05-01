ALTER TABLE attempt ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE attempt ADD COLUMN name TEXT;
ALTER TYPE attempt_status ADD VALUE 'kicked';
