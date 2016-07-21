ALTER TABLE states DROP CONSTRAINT flexid_and_flexidtype_not_null;
ALTER TABLE states DROP CONSTRAINT userid_or_flexid_not_null;
ALTER TABLE states ALTER COLUMN userid SET NOT NULL;
