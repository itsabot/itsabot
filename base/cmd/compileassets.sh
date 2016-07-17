#!/usr/bin/env bash

rm -rf $ABOT_PATH/public/*
mkdir -p $ABOT_PATH/public/{css,js,images}

# Concatenate files
cat $ABOT_PATH/assets/{vendor/,}js/*.js > $ABOT_PATH/public/js/main.js 2>/dev/null
cat $ABOT_PATH/assets/{vendor/,}css/*.css > $ABOT_PATH/public/css/main.css 2>/dev/null

# Create symbolic links to images
ln -f $ABOT_PATH/assets/images/* $ABOT_PATH/public/images/

# ln $ABOT_PATH/assets/images/* $ABOT_PATH/public/images
