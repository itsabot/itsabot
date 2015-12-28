CREATE TABLE knowledgeedges (
	userid INTEGER NOT NULL,
	startnodeid INTEGER NOT NULL,
	endnodeid INTEGER NOT NULL,
	nodepath INTEGER ARRAY NOT NULL,
	startnodeterm VARCHAR(255) NOT NULL,
	confidence INTEGER NOT NULL DEFAULT 50, -- between 0 and 100
	createdat TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
	updatedat TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
	PRIMARY KEY (startnodeid, endnodeid, nodepath, userid)
);
