# this file is meant to be sourced.

# usage: TAG MESSAGE
#
# NOTE: see put_tmp, put_over, and put functions below this one
#
# _put will print a right-aligned colorized TAG followed by the MESSAGE
# if TAG is empty, MESSAGE will be padded to align with the rest.
function _put {
	local tag=$1
	case "$tag" in
		"?") local color='\e[1;33m';;
		"ok") local color='\e[1;32m';;
		"warn") local color='\e[1;33m';;
		"err") local color='\e[1;31m';;
		"--") local color='\e[1;30m';;
		*) local color='\e[0m';;
	esac
	[ -n "$tag" ] && tag="[$tag]"
	printf "$color%8s\e[0m $2" "$tag"
}

# usage: TAG MESSAGE
# prints the TAG/MESSAGE followed by a newline. this is the generic option
# and does not allow for overwriting the terminal output
function put { _put "$1" "$2"; printf "\n"; }

# usage: TAG MESSAGE
# prints the TAG/MESSAGE but does not print a newline. this allows it to be
# replaced (likely via the put_over function)
function put_tmp { _put "$1" "$2"; }

# usage: TAG MESSAGE
# prints '\r', which clears the current line, before calling the generic put
function put_over { printf "\r"; put "$1" "$2"; }

# usage: HEADER CMD KIND(0=chk,1=warn,2=err) [MESSAGES...]
#        $1     $2  $3                       $4...
#
# NOTE: this function should not be called directly instead, use run_chk,
# run_warn, and run (which acts as run_err) functions below this one
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
	local ERROUT=/tmp/abot.err
	put_tmp "?" "$1"
	case "$3" in
		0) local TAG='--';;
		1) local TAG='warn';;
		2) local TAG='err';;
	esac
	bash -c "$2 &>$ERROUT" &>"$ERROUT"
	if [ "$?" -ne 0 ]; then
		put_over "$TAG" "$1"
		for ARG in "${@:4}"; do put '' "$ARG"; done
		if [ "$3" -eq 2 ]; then
			echo ""
			put "cmd" "$2\n"
			[ -n "$(cat $ERROUT)" ] && cat "$ERROUT"
			rm "$ERROUT"
			exit 1
		fi
		rm "$ERROUT"
		return 1
	else
		put_over "ok" "$1"
	fi
}

# usage: HEADER CMD [MESSAGES...]
#        $1     $2  $3...
function run_chk { _run "$1" "$2" 0 "${@:3}"; }
function run_warn { _run "$1" "$2" 1 "${@:3}"; }
function run { _run "$1" "$2" 2 "${@:3}"; }

# database connection parameter setup
# if the user specifies any database connection string at all, they must
# specify at least username and hostname. this avoids having to guess if a
# single string without ':" or '@' is referencing a username or hostname.
# password and port are optional, with port defaulting to 5432.
function parse_db_params {
	if [ -z "$1" ]; then
		DB_USER='postgres'
		DB_HOST='127.0.0.1'
		DB_PORT=5432
		return 0
	fi
	pattern='^([^:\/]+)(:([^@\/]*))?@([^:\/?]+)(:([0-9]+))?$'
	[[ "$1" =~ $pattern ]] || {
		echo -e "usage: $0 [username[:password]@host[:port=5432]]\n"
		echo "error: unable to parse database string"
		echo "please specify at least username and host"
		exit 1
	}
	DB_USER=${BASH_REMATCH[1]}
	DB_PASS=${BASH_REMATCH[3]}
	DB_HOST=${BASH_REMATCH[4]}
	DB_PORT=${BASH_REMATCH[6]:-"5432"}
}
