#!/usr/bin/env bash

source cmd/helper_functions.sh
parse_db_params "$1"

PSQL="psql -w -h '$DB_HOST' -p '$DB_PORT' -U '$DB_USER' -v ON_ERROR_STOP=1"
[ -n "$DB_PASS" ] && export PGPASSWORD=$DB_PASS

# begin setup
run "checking for psql binary" "which psql" \
	"please make sure 'psql' is in your path"

run "checking postgres connection" "$PSQL -c '\q'" \
	"unable to connect to postgres" \
	"please check your connection parameters..." \
	"(host: $DB_HOST, port $DB_PORT, user: $DB_USER, pass: $DB_PASS)" \
	"... and make sure postgres is running"

run_chk "checking for abot database" \
	"$PSQL -F'|' -tAc '\l' | grep -q '^abot|'" \
	"abot database missing. creating it"
[ "$?" -ne 0 ] && {
run "creating abot database" "$PSQL -c 'create database abot'" \
	"could not create abot database"
}

run_chk "checking for abot_test database" \
	"psql -F'|' -tAc '\l' | grep -q '^abot_test|'" \
	"abot_test database missing. creating it"
[ "$?" -ne 0 ] && {
run "creating abot_test database" "$PSQL -c 'create database abot_test'" \
	"could not create abot_test database"
}

MIGCMD='ls db/migrations/up/*.sql | sort | xargs -I{} --'
run_warn "running abot migrations" "$MIGCMD $PSQL -d abot -f {}" \
	"database migrations failed" \
	"if the database has already been migrated, you can ignore this message"

run_warn "running abot_test migrations" "$MIGCMD $PSQL -d abot_test -f {}" \
	"database migrations failed" \
	"if the database has already been migrated, you can ignore this message"

CITY_CNT=$(wc -l data/cities.csv | awk '{print $1}')
SEEDA="cat data/cities.csv | $PSQL"
SEEDB="COPY cities(name, countrycode) FROM stdin DELIMITER ',' CSV;"

PG_CNT=$(bash -c "$PSQL -d abot -tAc 'select count(*) from cities'")
run_warn "checking if abot database is seeded" \
	"[ '$PG_CNT' -ge '$CITY_CNT' ] || $SEEDA -d abot -c \"$SEEDB\"" \
	"if the database has already been seeded, you can ignore this message"

PG_CNT=$(bash -c "$PSQL -d abot_test -tAc 'select count(*) from cities'")
run_warn "checking if abot_test database is seeded" \
	"[ '$PG_CNT' -ge '$CITY_CNT' ] || $SEEDA -d abot_test -c \"$SEEDB\"" \
	"if the database has already been seeded, you can ignore this message"
