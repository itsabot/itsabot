CREATE TABLE knowledgequeries (
	id SERIAL,
	userid INTEGER NOT NULL,
	responseid INTEGER,
	term VARCHAR(35) NOT NULL,
	termlength INTEGER NOT NULL,
	trigram VARCHAR(255) NOT NULL,
	wordtype VARCHAR(7) NOT NULL,
	relation VARCHAR(255),
	active BOOLEAN NOT NULL DEFAULT TRUE,
	createdat TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
	UNIQUE (userid, term, trigram),
	PRIMARY KEY (id)
);
