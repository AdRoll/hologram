#!/bin/bash

source ${HOLOGRAM_DIR}/buildscripts/returncodes.sh

echo "Compiling for linux..."
GOOS=linux  go install github.com/AdRoll/hologram/... || exit ${ERRCOMPILE}

echo "Compiling for osx"
GOOS=darwin go install github.com/AdRoll/hologram/... || exit ${ERRCOMPILE}

echo "Running tests..."
go test -v github.com/AdRoll/hologram/... || exit ${ERRTEST}
