#!/usr/bin/env bash

set -e


ava_reload() {
	rm -rf public/*
	mkdir -p public/{css,js}

	echo reloading...
	jsfiles=$(find assets/{vendor/,}/js -type f -name "*.js")
	cssfiles=$(find assets/{vendor/,}css -type f -name "*.css")
	cat $jsfiles > public/js/main.js
	cat $cssfiles > public/css/main.css
	curl localhost:4200/_/cmd/reload
}

export -f ava_reload

fswatch -l 0.1 -0 -or assets | xargs -0 -n 1 -I {} -- bash -c ava_reload
