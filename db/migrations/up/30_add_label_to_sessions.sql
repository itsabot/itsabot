ALTER TABLE sessions ADD COLUMN label VARCHAR(255);
ALTER TABLE sessions ADD UNIQUE (userid, label);
