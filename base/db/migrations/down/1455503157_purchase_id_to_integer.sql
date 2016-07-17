ALTER TABLE purchases DROP CONSTRAINT purchases_pkey;
ALTER TABLE purchases DROP COLUMN id;
ALTER TABLE purchases ADD COLUMN id UUID DEFAULT uuid_generate_v4() NOT NULL;
ALTER TABLE purchases ADD PRIMARY KEY (id);
