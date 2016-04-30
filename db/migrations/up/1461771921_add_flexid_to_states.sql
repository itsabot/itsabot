ALTER TABLE states ADD COLUMN flexid VARCHAR(255);
ALTER TABLE states ADD COLUMN flexidtype INTEGER;
ALTER TABLE states ADD UNIQUE (flexid, flexidtype, pluginname, key);
ALTER TABLE states ALTER COLUMN userid DROP NOT NULL;

-- From http://stackoverflow.com/a/15180123
ALTER TABLE states ADD CONSTRAINT userid_or_flexid_not_null CHECK (
	(userid IS NOT NULL)::INTEGER + (flexid IS NOT NULL)::INTEGER = 1
)
