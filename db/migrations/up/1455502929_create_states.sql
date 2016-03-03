-- TODO
-- Enable branching states with a users_states join table. Each plugin
-- maintains a state for each user. Then a user can build up a state in one
-- plugin, perform some task in another plugin, and RETURN to the first plugin.
-- First a plugin should check the most recently set state for the user, but if
-- something is missing, (e.g. a list of recommendations), then it should check
-- the last state of other plugins by key name (e.g.  "recommendations"),
-- ordered by createdat DESC. This will enable "piping" of commands from plugin
-- to plugin, allowing the user to jump seamlessly between plugins without
-- losing state, keeping with their expectations.
--
-- Perhaps the best way to accomplish that is for plugins to implement a
-- variety of pre-set interfaces, similar to intents in Android. I can say,
-- `memory.GetRecommendations(dt.Restaurant)` and it should find the plugins
-- with that capability registered (i.e. responds to intent). 

CREATE TABLE states (
	id SERIAL,
	userid INTEGER NOT NULL,
	pkgname VARCHAR(255) NOT NULL,
	state JSONB,
	updatedat TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
	PRIMARY KEY (id)
);
