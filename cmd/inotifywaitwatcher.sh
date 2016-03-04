#!/usr/bin/env bash

set -e

inotifywait -m -r -e modify,attrib,close_write,move,create,delete assets | 
	while read file; do
		curl localhost:4200/_/cmd/reload
	done
