CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE purchases (
	id UUID DEFAULT uuid_generate_v4(),
	userid INTEGER NOT NULL,
	vendorid INTEGER NOT NULL,
	shippingaddressid INTEGER,
	products VARCHAR(255)[] NOT NULL,
	tax INTEGER NOT NULL,
	shipping INTEGER NOT NULL,
	total INTEGER NOT NULL, -- what the customer paid
	avafee INTEGER NOT NULL,
	creditcardfee INTEGER NOT NULL, -- 2.9% + 30¢
	transferfee INTEGER NOT NULL, -- 0.5%
	vendorpayout INTEGER NOT NULL, -- amount the vendor will receive:
	--   subtotal + tax + shipping
	-- - total*2.9% - 30¢ (credit card fee)
	-- - total*5% (Ava's fee)
	-- = payout
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
