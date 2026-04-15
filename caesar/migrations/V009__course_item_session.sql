ALTER TABLE course_item DROP COLUMN order_index;
ALTER TABLE course_item RENAME COLUMN ref_id TO object_id;
