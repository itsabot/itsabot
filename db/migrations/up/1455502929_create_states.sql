-- TODO
-- Enable branching states with a users_states join table. Each package
-- maintains a state for each user. Then a user can build up a state in one
-- package, perform some task in another package, and RETURN to the first
-- package. First a package should check the most recently set state for the
-- user, but if something is missing, (e.g. a list of recommendations), then it
-- should check the last state of other packages by key name (e.g.
-- "recommendations"), ordered by createdat DESC. This will enable "piping" of
-- commands from package to package, allowing the user to jump seamlessly
-- between packages without losing state, keeping with their expectations.
--
-- Perhaps the best way to accomplish that is for packages to implement a
-- variety of pre-set interfaces, similar to intents in Android. I can say,
-- `memory.GetRecommendations(dt.Restaurant)` and it should find the packages
-- with that capability registered (i.e. responds to intent). 

CREATE TABLE states (
	id SERIAL,
	userid INTEGER NOT NULL,
	pkgname VARCHAR(255) NOT NULL,
	state JSONB,
	updatedat TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
	PRIMARY KEY (id)
);
