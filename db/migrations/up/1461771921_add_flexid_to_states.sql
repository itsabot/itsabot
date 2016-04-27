ALTER TABLE states ADD COLUMN flexid VARCHAR(255);
ALTER TABLE states ADD COLUMN flexidtype INTEGER;
ALTER TABLE states ADD UNIQUE (flexid, flexidtype, pluginname, key);
