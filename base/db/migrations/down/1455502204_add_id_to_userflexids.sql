ALTER TABLE userflexids DROP CONSTRAINT userflexids_pkey;
ALTER TABLE userflexids ADD PRIMARY KEY (userid, flexid);
ALTER TABLE userflexids DROP COLUMN id;
