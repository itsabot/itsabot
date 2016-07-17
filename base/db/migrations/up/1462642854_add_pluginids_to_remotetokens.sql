ALTER TABLE remotetokens DROP COLUMN pluginid;
ALTER TABLE remotetokens ADD COLUMN pluginids INTEGER[] NOT NULL;
