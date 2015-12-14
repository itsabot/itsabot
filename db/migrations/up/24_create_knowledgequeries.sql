CREATE TABLE knowledgequeries (
	id SERIAL,
	userid INTEGER NOT NULL,
	term VARCHAR(35) NOT NULL,
	trigram VARCHAR(255) NOT NULL,
	wordtype VARCHAR(7) NOT NULL,
	relation VARCHAR(255),
	active BOOLEAN DEFAULT TRUE,
	PRIMARY KEY (id)
);
