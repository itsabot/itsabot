ALTER TABLE states ALTER COLUMN userid DROP NOT NULL;
ALTER TABLE states ADD CONSTRAINT userid_or_flexid_not_null CHECK (
	(userid IS NOT NULL)::INTEGER + (flexid IS NOT NULL)::INTEGER = 1
);
ALTER TABLE states ADD CONSTRAINT flexid_and_flexidtype_not_null CHECK (
	(flexidtype IS NOT NULL)::INTEGER + (flexid IS NOT NULL)::INTEGER = 2 OR
	(flexidtype IS NOT NULL)::INTEGER + (flexid IS NOT NULL)::INTEGER = 0
);
