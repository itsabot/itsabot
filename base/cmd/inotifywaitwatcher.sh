#!/usr/bin/env bash

set -e

inotifywait -m -r -e modify assets |
	while read file; do
		echo reloading...
		curl localhost:4200/_/cmd/ws/reload
	done
