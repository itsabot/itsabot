CREATE TABLE addresses (
	id SERIAL,
	userid INTEGER NOT NULL,
	name VARCHAR(255) NOT NULL,
	line1 VARCHAR(255) NOT NULL,
	line2 VARCHAR(255) NOT NULL,
	city VARCHAR(255) NOT NULL,
	state VARCHAR(255) NOT NULL,
	country VARCHAR(255) NOT NULL,
	zip VARCHAR(20) NOT NULL, -- full zip, either zip5+4 or international
	PRIMARY KEY (id)
);
