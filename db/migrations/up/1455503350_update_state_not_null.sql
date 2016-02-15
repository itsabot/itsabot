ALTER TABLE states DROP COLUMN state;
ALTER TABLE states ADD COLUMN key VARCHAR(255);
ALTER TABLE states ADD COLUMN value bytea;
ALTER TABLE states ADD UNIQUE (userid, pkgname, key);
