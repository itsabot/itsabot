#!/bin/bash

set -e

go install $(go list ./... | grep -v /vendor/)
