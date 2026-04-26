ALTER TABLE question
DROP COLUMN IF EXISTS image_link;

ALTER TABLE question
ADD COLUMN image_id UUID;

CREATE INDEX idx_question_image_id ON question(image_id);
