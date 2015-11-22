CREATE TABLE sessions (
	userid INTEGER,
	token VARCHAR(255),
	PRIMARY KEY (userid, token)
);
