ALTER TABLE remotetokens ADD COLUMN pluginid INTEGER NOT NULL;
ALTER TABLE remotetokens DROP COLUMN pluginids;
