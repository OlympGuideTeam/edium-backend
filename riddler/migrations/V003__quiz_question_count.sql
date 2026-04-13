ALTER TABLE quiz_template ADD COLUMN question_count INTEGER NOT NULL DEFAULT 0;

CREATE OR REPLACE FUNCTION update_quiz_question_count() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE quiz_template SET question_count = question_count + 1 WHERE id = NEW.quiz_template_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE quiz_template SET question_count = question_count - 1 WHERE id = OLD.quiz_template_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_question_count
    AFTER INSERT OR DELETE ON question
    FOR EACH ROW EXECUTE FUNCTION update_quiz_question_count();
