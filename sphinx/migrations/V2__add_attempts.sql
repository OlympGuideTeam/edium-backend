-- Счётчик попыток обработки задачи (нужен для восстановления после прерывания VM)
ALTER TABLE generation_task
    ADD COLUMN attempts INT NOT NULL DEFAULT 0;
