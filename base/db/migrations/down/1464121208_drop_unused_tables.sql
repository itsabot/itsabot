CREATE TABLE addresses (
	id SERIAL,
	userid INTEGER NOT NULL,
	name VARCHAR(255) NOT NULL,
	line1 VARCHAR(255) NOT NULL,
	line2 VARCHAR(255) NOT NULL,
	city VARCHAR(255) NOT NULL,
	state VARCHAR(255) NOT NULL,
	country VARCHAR(255) NOT NULL,
	zip VARCHAR(20) NOT NULL,
	zip5 VARCHAR(5),
	zip4 VARCHAR(4),
	PRIMARY KEY (id)
);
CREATE TABLE authorizations (
	id SERIAL,
	authmethod INTEGER NOT NULL,
	createdat TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
	authorizedat TIMESTAMP,
	attempts INTEGER DEFAULT 0 NOT NULL,
	PRIMARY KEY (id)
);
CREATE TABLE emails (
	messageid INTEGER NOT NULL,
	deliveredto TEXT,
	replyto TEXT,
	PRIMARY KEY (messageid)
);
CREATE TABLE locations (
	id SERIAL,
	name VARCHAR(255),
	lat FLOAT,
	lon FLOAT,
	PRIMARY KEY (id)
);
CREATE TABLE inputs (
	id SERIAL,
	userid INTEGER,
	sentence VARCHAR(255),
	plugin VARCHAR(255),
	route VARCHAR(255),
	createdat TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
	trainingid INTEGER,
	sentenceannotated VARCHAR(255),
	abotsent BOOLEAN DEFAULT TRUE NOT NULL,
	PRIMARY KEY (id)
);
CREATE EXTENSION "uuid-ossp";
CREATE TABLE purchases (
	id UUID DEFAULT uuid_generate_v4(),
	userid INTEGER NOT NULL,
	vendorid INTEGER NOT NULL,
	shippingaddressid INTEGER,
	products VARCHAR(255)[] NOT NULL,
	tax INTEGER NOT NULL,
	shipping INTEGER NOT NULL,
	total INTEGER NOT NULL,
	avafee INTEGER NOT NULL,
	creditcardfee INTEGER NOT NULL,
	transferfee INTEGER NOT NULL,
	vendorpayout INTEGER NOT NULL,
	createdat TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
	emailssentat TIMESTAMP,
	vendorpaidat TIMESTAMP,
	deliveryexpectedat TIMESTAMP,
	PRIMARY KEY (id)
);
CREATE TABLE vendors (
	id SERIAL,
	businessname VARCHAR(255) NOT NULL,
	contactname VARCHAR(255) NOT NULL,
	contactemail VARCHAR(255) NOT NULL,
	contactphone VARCHAR(255) NOT NULL,
	PRIMARY KEY (id)
);
CREATE TABLE responses (
	id SERIAL,
	userid INTEGER NOT NULL,
	inputid INTEGER NOT NULL,
	sentence VARCHAR(255) NOT NULL,
	route VARCHAR(255) NOT NULL,
	state JSONB,
	createdat TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
	PRIMARY KEY (id)
);
ALTER TABLE messages ADD COLUMN sentenceannotated VARCHAR(255);
ALTER TABLE messages ADD COLUMN trainingid INTEGER;
