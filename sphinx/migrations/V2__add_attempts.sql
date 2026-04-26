ALTER TABLE generation_task
    ADD COLUMN attempts INT NOT NULL DEFAULT 0;
