CREATE EXTENSION pg_trgm;
CREATE TABLE contacts (
	id SERIAL,
	name VARCHAR(255) NOT NULL,
	email VARCHAR(255),
	phone VARCHAR(255),
	userid INTEGER NOT NULL,
	createdat TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
	updatedat TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
	UNIQUE (userid, name, email, phone),
	PRIMARY KEY (id)
);
CREATE INDEX contacts_trgm_idx ON contacts USING GIN (
	name gin_trgm_ops,
	email gin_trgm_ops,
	phone gin_trgm_ops
);
