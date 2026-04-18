CREATE TYPE quiz_source AS ENUM ('library', 'course');

ALTER TABLE quiz_template
    ADD COLUMN source quiz_source NOT NULL DEFAULT 'library',
    DROP COLUMN is_draft;
