#!/usr/bin/env bash

PFX='         '
ERROUT=/tmp/abot.err
ABOTPKG='github.com/itsabot/abot'
DMPKG='github.com/robfig/glock'
PORT="4200"
ABOT_ENV="development"
ABOT_URL="http://localhost:$PORT"
ABOT_SECRET=$(< /dev/urandom LC_CTYPE=C tr -dc _A-Z-a-z-0-9 | head -c${1:-64})

# usage: HEADER CMD KIND(0=chk,1=warn,2=err) [MESSAGES...]
#        $1     $2  $3                       $4...
#
# NOTE: this function should not be called directly
#       instead, use run_chk, run_warn, and run (which act as a run_err)
#       functions below this one
#
# _run will display [?] HEADER, then execute CMD
# if CMD is successful, the previous HEADER is overwritten with [ok] HEADER
# if CMD fails:
#   if KIND is err:
#     the previous HEADER is overwritten with [err] HEADER
#     MESSAGES are printed
#     CMD's stdout/stderr is displayed and the script exits
#   if KIND is warn:
#     the previous HEADER is overwritten with [warn] HEADER
#     MESSAGES are printed
#     the function returns 1
#   if KIND is chk"
#     the previous HEADER is overwritten with [--] HEADER
#     MESSAGES are printed
#     the function returns 1
#
# finally, bash escaping rules can be tricky for the CMD's passed to this
# function. have fun
function _run {
	local TAG='     \e[1;33m[?]\e[0m'
	printf "$TAG $1"
	case "$3" in
		0) TAG='    \e[1;30m[--]\e[0m'; bash -c "$2 &>/dev/null" ;;
		1) TAG='  \e[1;33m[warn]\e[0m'; bash -c "$2 &>/dev/null" ;;
		2) TAG='   \e[1;31m[err]\e[0m'; bash -c "$2 &>$ERROUT" ;;
	esac
	if [ "$?" -ne 0 ]; then
		printf "\r$TAG $1\n"
		if [ "$3" -eq 0 ]; then
			for ARG in "${@:4}"; do echo "$PFX$ARG"; done
			return 1
		elif [ "$3" -eq 1 ]; then
			for ARG in "${@:4}"; do echo "$ARG"; done
			[ "$#" -gt 3 ] && echo ""
			return 1
		fi
		echo -e "\nfailed cmd:\n${PFX}$2\n"
		for ARG in "${@:4}"; do echo "$ARG"; done
		[ "$#" -gt 3 ] && echo ""
		if [ -n "$(cat $ERROUT)" ]; then
			cat "$ERROUT"
		fi
		rm "$ERROUT"
		exit 1
	else
		printf "\r\e[1;32m    [ok]\e[0m $1\n"
	fi
}

# usage: HEADER CMD [MESSAGES...]
#        $1     $2  $3...
function run_chk { _run "$1" "$2" 0 "${@:3}"; }
function run_warn { _run "$1" "$2" 1 "${@:3}"; }
function run { _run "$1" "$2" 2 "${@:3}"; }

# begin setup
run "checking for go binary" "which go" "please make sure 'go' is in your path"
run "checking GOPATH" "[ -n '$GOPATH' ]" "GOPATH is not set"
run "installing dependency manager" "go get '$DMPKG'"
run "syncing dependencies" "glock sync '$ABOTPKG'"
run "installing glock hook" "glock install '$ABOTPKG'"
run "installing abot" "go install '$ABOTPKG'"

run "checking for postgres" "ps ax -o comm | grep -q '^postgres'" \
	"please start postgres to continue"

run "checking for psql binary" "which psql" \
	"please make sure 'psql' is in your path"

run "checking for postgres user" \
	"psql -U postgres -tAc '\du' | grep -q '^postgres|'" \
	"please create a postgres user named 'postgres'"

run_chk "checking for abot database" \
	"psql -U postgres -tAc '\l' | grep -q '^abot|'" \
	"abot database missing. creating it"
[ "$?" -ne 0 ] && {
run "creating abot database" "createdb -U postgres -O postgres abot" \
	"could not create database"
}

run_chk "checking for abot_test database" \
	"psql -U postgres -tAc '\l' | grep -q '^abot_test|'" \
	"abot_test database missing. creating it"
[ "$?" -ne 0 ] && {
run "creating abot_test database" "createdb -U postgres -O postgres abot_test" \
	"could not create database"
}

MIGCMD='ls db/migrations/up/*.sql | sort | xargs -I{} -- \
	psql -v ON_ERROR_STOP=1 -U postgres'
run_warn "running abot migrations" "$MIGCMD -d abot -f {}" \
	"database migrations failed" \
	"if the database has already been migrated, you can ignore this message"

run_warn "running abot_test migrations" "$MIGCMD -d abot_test -f {}" \
	"database migrations failed" \
	"if the database has already been migrated, you can ignore this message"

CITY_CNT=$(wc -l data/cities.csv | awk '{print $1}')
SEEDA="cat data/cities.csv | psql -U postgres -d"
SEEDB="COPY cities(name, countrycode) FROM stdin DELIMITER ',' CSV;"

PG_CNT=$(psql -tAc -U postgres -d abot -c 'select count(*) from cities')
run_warn "checking if abot database is seeded" \
	"[ '$PG_CNT' -ge '$CITY_CNT' ] || $SEEDA abot -c \"$SEEDB\"" \
	"if the database has already been seeded, you can ignore this message"

PG_CNT=$(psql -tAc -U postgres -d abot_test -c 'select count(*) from cities')
run_warn "checking if abot_test database is seeded" \
	"[ \"$PG_CNT\" -ge \"$CITY_CNT\" ] || $SEEDA abot_test -c \"$SEEDB\"" \
	"if the test database has already been seeded, you can ignore this message"

# note: the ';' at the end of this command is important. do not remove.
run "generating environment file" "echo 'PORT=$PORT
ABOT_ENV=$ABOT_ENV
ABOT_URL=$ABOT_URL
ABOT_SECRET=$ABOT_SECRET' > abot.env;"

run "installing abot plugins" "abot plugin install"

rm "$ERROUT"

printf "\n*\e[1;32m [complete]\e[0m"
printf " ***********************************************************\n\n"

echo "to boot abot:
    1. run 'abot server'
    2. open a web browser to $ABOT_URL

you'll want to sign up to create a user account next"
