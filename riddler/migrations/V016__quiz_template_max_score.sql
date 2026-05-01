ALTER TABLE quiz_template ADD COLUMN max_score INTEGER NOT NULL DEFAULT 0;

UPDATE quiz_template qt
SET max_score = COALESCE((
    SELECT SUM(q.max_score) FROM question q WHERE q.quiz_template_id = qt.id
), 0);

CREATE OR REPLACE FUNCTION update_quiz_max_score() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE quiz_template SET max_score = max_score + NEW.max_score WHERE id = NEW.quiz_template_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE quiz_template SET max_score = max_score - OLD.max_score WHERE id = OLD.quiz_template_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_question_max_score
    AFTER INSERT OR DELETE ON question
    FOR EACH ROW EXECUTE FUNCTION update_quiz_max_score();
