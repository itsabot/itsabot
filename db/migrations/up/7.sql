ALTER TABLE trainings ADD maxassignments INTEGER DEFAULT 1 NOT NULL;
ALTER TABLE inputs ADD trainingid INTEGER;
ALTER TABLE inputs ADD sentenceannotated VARCHAR(255);
