#!/usr/bin/env bash

set -e

echo "Verifying PostgreSQL is running... "
if ! ps ax | grep 'postgres' > /dev/null
then
	echo "PostgreSQL not running. Please start Postgres to continue."
	exit 1
fi

echo "Confirming Postgres user exists..."
if ! psql -U postgres postgres -tAc "SELECT '' FROM pg_roles WHERE rolname='postgres'" > /dev/null
then
	echo "Please create a PostgreSQL user named postgres to continue. Google \"createuser postgres\""
fi

echo "Creating abot database..."
if ! createdb -U postgres abot -O postgres &> /dev/null
then
	echo "WARN: could not create abot database. If you already created one, you can ignore this message."
fi

echo "Creating abot_test database..."
if ! createdb -U postgres abot_test -O postgres &> /dev/null
then
	echo "WARN: could not create abot_test database. If you already created one, you can ignore this message."
fi

echo "Migrating databases..."
if ! cmd/migrateup.sh &> /dev/null
then
	echo "WARN: could not migrate db."
fi

echo "Updating environment variables"
if [ -f ~/.bash_profile ];
then
	FILE=$HOME/.bash_profile
else
	FILE=$HOME/.bashrc
fi

# Delete old lines
sed -i='' -n '/\n\n# Added by Abot via /!p' $FILE
sed -i='' -n '/\n# Added by Abot via /!p' $FILE
sed -i='' -n '/# Added by Abot via /!p' $FILE
sed -i='' -n '/export PORT=/!p' $FILE
sed -i='' -n '/export ABOT_URL=/!p' $FILE
sed -i='' -n '/export ABOT_ENV=/!p' $FILE
sed -i='' -n '/export ABOT_SECRET=/!p' $FILE

# Generate ABOT_SECRET used for validating cookie values
SECRET=$(< /dev/urandom tr -dc _A-Z-a-z-0-9 | head -c${1:-64};echo;)

# Append environment variables
cat <<EOT >> $FILE

# Added by Abot via setup.sh 
export PORT="4200"
export ABOT_ENV="development"
export ABOT_URL="http://localhost:4200"
export ABOT_SECRET="$SECRET"
EOT

# Source new env vars for current terminal
export PORT="4200"
export ABOT_ENV="development"
export ABOT_URL="http://localhost:4200"
export ABOT_SECRET="$SECRET"

echo "Fetching dependencies..."
go get github.com/robfig/glock &>/dev/null
glock sync github.com/itsabot/abot &>/dev/null
glock install github.com/itsabot/abot &>/dev/null

echo "Installing Abot..."
go install
abot plugin install

echo "Seeding database..."
if ! cat data/cities.csv | psql -U postgres -d abot -c "COPY cities(name, countrycode) FROM stdin DELIMITER ',' CSV;" &> /dev/null
then
	echo "WARN: could not seed abot database. If you already seeded it, you can ignore this message."
fi
if ! cat data/cities.csv | psql -U postgres -d abot_test -c "COPY cities(name, countrycode) FROM stdin DELIMITER ',' CSV;" &> /dev/null
then
	echo "WARN: could not seed abot_test database. If you already seeded it, you can ignore this message."
fi

echo "To boot Abot, run \"abot server\" and open a web browser to localhost:4200."
echo "You'll want to sign up to create a user account next."
