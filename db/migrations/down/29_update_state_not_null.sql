ALTER TABLE states DROP CONSTRAINT states_userid_pkgname_key_key;
ALTER TABLE states DROP COLUMN key;
ALTER TABLE states DROP COLUMN value;
ALTER TABLE states ADD COLUMN state jsonb;
