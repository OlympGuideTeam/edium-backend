-- Среда, из которой пришёл запрос: 'prod' или 'test'
ALTER TABLE generation_task
    ADD COLUMN env VARCHAR(10) NOT NULL DEFAULT 'prod';

ALTER TABLE generation_result
    ADD COLUMN env VARCHAR(10) NOT NULL DEFAULT 'prod';
