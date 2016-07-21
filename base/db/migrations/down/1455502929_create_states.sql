ALTER TABLE responses DROP COLUMN stateid;
ALTER TABLE responses ADD COLUMN state JSONB;
DROP TABLE states;
