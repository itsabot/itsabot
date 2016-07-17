#!/usr/bin/env bash

set -e


ava_reload() {
	echo reloading...
	curl localhost:4200/_/cmd/ws/reload
}

export -f ava_reload

fswatch -l 0.1 -0 -or assets | xargs -0 -n 1 -I {} -- bash -c ava_reload
