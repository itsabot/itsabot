#!/usr/bin/env bash

rm -rf public/*
mkdir -p public/{css,js,images}

# Concatenate files
cat {plugins/*/,}assets/{vendor/,}js/*.js > public/js/main.js 2>/dev/null
cat {plugins/*/,}assets/{vendor/,}css/*.css > public/css/main.css 2>/dev/null

# Create symbolic links to images
# ln -s plugins/*/public/images/* public/images
# ln -s assets/images/* public/images

ln plugins/*/public/images/* public/images
ln assets/images/* public/images
