#!/bin/bash
# Copyright IBM Corp. 2017, 2025
# SPDX-License-Identifier: MIT


set -e
echo "" > coverage.txt

for d in $(go list ./... | grep -v vendor); do
    go test -coverprofile=profile.out -covermode=atomic $d
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done
