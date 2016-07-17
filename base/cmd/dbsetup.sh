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

run_chk "checking for database" \
	"$PSQL -F'|' -tAc '\l' | grep -q '^$DB_NAME|'" \
	"$DB_NAME database missing. creating it"
[ "$?" -ne 0 ] && {
run "creating $DB_NAME database" "$PSQL -c 'create database $DB_NAME'" \
	"could not create $DB_NAME database"
}

run_chk "checking for ${DB_NAME}_test database" \
	"$PSQL -F'|' -tAc '\l' | grep -q '^$DB_NAME}_test|'" \
	"${DB_NAME}_test database missing. creating it"
[ "$?" -ne 0 ] && {
run "creating ${DB_NAME}_test database" "$PSQL -c 'create database ${DB_NAME}_test'" \
	"could not create ${DB_NAME}_test database"
}

MIGCMD='ls db/migrations/up/*.sql | sort | xargs -I{} --'
run_warn "running $DB_NAME migrations" "$MIGCMD $PSQL -d $DB_NAME -f {}" \
	"database migrations failed" \
	"if the database has already been migrated, you can ignore this message"

run_warn "running ${DB_NAME}_test migrations" "$MIGCMD $PSQL -d ${DB_NAME}_test -f {}" \
	"database migrations failed" \
	"if the database has already been migrated, you can ignore this message"

CITY_CNT=$(wc -l data/cities.csv | awk '{print $1}')
SEEDA="cat data/cities.csv | $PSQL"
SEEDB="COPY cities(name, countrycode) FROM stdin DELIMITER ',' CSV;"

PG_CNT=$(bash -c "$PSQL -d $DB_NAME -tAc 'select count(*) from cities'")
run_warn "checking if $DB_NAME database is seeded" \
	"[ '$PG_CNT' -ge '$CITY_CNT' ] || $SEEDA -d $DB_NAME -c \"$SEEDB\"" \
	"if the database has already been seeded, you can ignore this message"

PG_CNT=$(bash -c "$PSQL -d ${DB_NAME}_test -tAc 'select count(*) from cities'")
run_warn "checking if ${DB_NAME}_test database is seeded" \
	"[ '$PG_CNT' -ge '$CITY_CNT' ] || $SEEDA -d ${DB_NAME}_test -c \"$SEEDB\"" \
	"if the database has already been seeded, you can ignore this message"
