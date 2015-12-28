CREATE TABLE knowledgenodes (
	id SERIAL,
	userid INTEGER NOT NULL,
	term VARCHAR(35) NOT NULL,
	termstem VARCHAR(35) NOT NULL,
	termlength INTEGER NOT NULL,
	termtype INTEGER NOT NULL, -- 1: commandNode, 2: objectNode
	relation VARCHAR(255),
	confidence INTEGER DEFAULT 50 NOT NULL, -- between 0 and 100
	createdat TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
	UNIQUE (userid, term),
	PRIMARY KEY (id)
);
