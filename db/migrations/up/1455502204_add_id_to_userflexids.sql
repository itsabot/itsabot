ALTER TABLE userflexids ADD COLUMN id SERIAL;
ALTER TABLE userflexids DROP CONSTRAINT userflexids_pkey;
ALTER TABLE userflexids ADD PRIMARY KEY (id);
