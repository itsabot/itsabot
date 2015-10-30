ALTER TABLE responses ADD feedbackid INTEGER;

CREATE TABLE feedbacks (
	id SERIAL,
	sentence VARCHAR(255),
	-- -1 negative, 0 neutral, 1 positive
	sentiment INTEGER NOT NULL,
	createdat TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
	PRIMARY KEY (id)
);
