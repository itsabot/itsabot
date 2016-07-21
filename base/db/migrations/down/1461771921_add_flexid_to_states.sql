ALTER TABLE states ALTER COLUMN userid SET NOT NULL;
ALTER TABLE states DROP CONSTRAINT userid_or_flexid_not_null;
ALTER TABLE states DROP CONSTRAINT states_flexid_flexidtype_pluginname_key_key;
ALTER TABLE states DROP COLUMN flexidtype;
ALTER TABLE states DROP COLUMN flexid;
