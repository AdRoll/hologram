#!/bin/bash

export GIT_TAG=$(git describe --tags --long | sed 's/-/\./' | sed 's/-g/-/' | sed 's/-/~/')

if [ "$1" != "--no-compile" ]; then
    compile_hologram.sh || exit 1
fi

mkdir -p /hologram-build/{server,agent}/root/usr/local/bin
mkdir -p /hologram-build/{server,agent}/root/etc/hologram
mkdir -p /hologram-build/{server,agent}/scripts/
mkdir -p /hologram-build/{server,agent}/root/etc/init.d/

cp ${HOLOGRAM_DIR}/config/agent.json /hologram-build/agent/root/etc/hologram/agent.json
cp ${BIN_DIR}/hologram-cli /hologram-build/agent/root/usr/local/bin/
cp ${BIN_DIR}/hologram-agent /hologram-build/agent/root/usr/local/bin/
cp ${BIN_DIR}/hologram-authorize /hologram-build/agent/root/usr/local/bin/

cp ${HOLOGRAM_DIR}/agent/support/debian/after-install.sh /hologram-build/agent/scripts/
cp ${HOLOGRAM_DIR}/agent/support/debian/before-remove.sh /hologram-build/agent/scripts/
cp ${HOLOGRAM_DIR}/agent/support/debian/init.sh /hologram-build/agent/root/etc/init.d/hologram-agent

cp ${HOLOGRAM_DIR}/config/server.json /hologram-build/server/root/etc/hologram/server.json

cp ${BIN_DIR}/hologram-cli /hologram-build/server/root/usr/local/bin/
cp ${BIN_DIR}/hologram-server /hologram-build/server/root/usr/local/bin/
cp ${BIN_DIR}/hologram-authorize /hologram-build/server/root/usr/local/bin/

cp ${HOLOGRAM_DIR}/server/after-install.sh /hologram-build/server/scripts/
cp ${HOLOGRAM_DIR}/server/before-remove.sh /hologram-build/server/scripts/

cp ${HOLOGRAM_DIR}/server/support/hologram.init.sh /hologram-build/server/root/etc/init.d/hologram-server

ARTIFACTS_DIR=${HOLOGRAM_DIR}/artifacts
mkdir -p ${ARTIFACTS_DIR}

cd /hologram-build/agent
fpm -f -s dir -t deb -n hologram-agent -v ${GIT_TAG}  --after-install /hologram-build/agent/scripts/after-install.sh  --before-remove /hologram-build/agent/scripts/before-remove.sh  --config-files /etc/hologram/agent.json  -C /hologram-build/agent/root  -p ${ARTIFACTS_DIR}/hologram-${GIT_TAG}.deb        -a amd64 .  || exit 1

cd /hologram-build/server
fpm -f -s dir -t deb -n hologram-server -v ${GIT_TAG} --after-install /hologram-build/server/scripts/after-install.sh --before-remove /hologram-build/server/scripts/before-remove.sh --config-files /etc/hologram/server.json -C /hologram-build/server/root -p ${ARTIFACTS_DIR}/hologram-server-${GIT_TAG}.deb -a amd64 . || exit 1
