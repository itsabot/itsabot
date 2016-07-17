CREATE TABLE settings (
	name VARCHAR(255) NOT NULL,
	value VARCHAR(255) NOT NULL,
	pluginname VARCHAR(255) NOT NULL,
	PRIMARY KEY (pluginname, name)
);
