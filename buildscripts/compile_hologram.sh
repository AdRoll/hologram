#!/bin/bash

source ${HOLOGRAM_DIR}/buildscripts/returncodes.sh

export GIT_TAG=$(git describe --tags --long)

compile() {
    echo "Compiling for $1/$2..."
    GOOS=$1 GOARCH=$2 CGO_ENABLED=0 go build -trimpath \
        -ldflags="-X 'main.Version=${GIT_TAG}'" \
        -o "${BIN_DIR}/$1_$2/" github.com/AdRoll/hologram/... || exit ${ERRCOMPILE}
}

compile linux amd64
compile darwin amd64
compile darwin arm64

echo "Combining the darwin builds into universal binaries..."
mkdir -p ${BIN_DIR}/darwin_universal
for binary in ${BIN_DIR}/darwin_arm64/* ; do
    name=$(basename ${binary})
    llvm-lipo -create -output ${BIN_DIR}/darwin_universal/${name} \
        ${BIN_DIR}/darwin_amd64/${name} \
        ${BIN_DIR}/darwin_arm64/${name} || exit ${ERRCOMPILE}
done

echo "Running tests..."
go test -v github.com/AdRoll/hologram/... || exit ${ERRTEST}
