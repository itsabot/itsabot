#!/usr/bin/env bash

set -e

mkdir -p public/js public/css
cat {plugins/*/,}assets/{vendor/,}js/*.js > public/js/main.js 2>/dev/null
cat {plugins/*/,}assets/{vendor/,}css/*.css > public/css/main.css 2>/dev/null
cp -r plugins/*/public/images/. public/images
cp -r assets/images/. public/images
