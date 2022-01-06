#!/bin/bash

source ${HOLOGRAM_DIR}/buildscripts/returncodes.sh

# go get -u github.com/kardianos/govendor

export GIT_TAG=$(git describe --tags --long)

echo "Compiling for linux..."
GOOS=linux  go install -ldflags="-X 'main.Version=${GIT_TAG}'" github.com/AdRoll/hologram/... || exit ${ERRCOMPILE}

echo "Compiling for osx"
GOOS=darwin go install -ldflags="-X 'main.Version=${GIT_TAG}'" github.com/AdRoll/hologram/... || exit ${ERRCOMPILE}

echo "Running tests..."
go test -v github.com/AdRoll/hologram/... || exit ${ERRTEST}
