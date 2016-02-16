#!/usr/bin/env bash

set -e

ls -r db/migrations/down/*.sql | xargs -I{} -- psql -U postgres ava -f {}
