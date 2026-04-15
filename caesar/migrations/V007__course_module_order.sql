ALTER TABLE course_module
    ADD COLUMN order_index INTEGER NOT NULL DEFAULT 0;

WITH ranked AS (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY course_id ORDER BY created_at) AS rn
    FROM course_module
)
UPDATE course_module
SET order_index = ranked.rn
FROM ranked
WHERE course_module.id = ranked.id;
