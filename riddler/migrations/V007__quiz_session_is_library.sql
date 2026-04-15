ALTER TABLE quiz_template ADD COLUMN library_session_id UUID REFERENCES quiz_session(id);
