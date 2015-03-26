#!/bin/bash

if [ "$1" == "--deps" ]; then
    install_deps.sh || exit 1
fi

echo "Running tests..."
go test -v ./... || exit 1

echo "Compiling for linux..."
GOOS=linux  go install ./... || exit 1

echo "Compiling for osx"
GOOS=darwin go install ./... || exit 1
