#!/bin/bash

source ${HOLOGRAM_DIR}/buildscripts/returncodes.sh
cd ${HOLOGRAM_DIR} && export GIT_TAG=$(git describe --tags --long | sed 's/-/\./' | sed 's/-g/-/' | sed 's/-/~/')

if [ "$1" != "--no-compile" ]; then
    compile_hologram.sh || exit $?
fi

mkdir -p /hologram-build/{server,agent}/root/usr/local/bin
mkdir -p /hologram-build/{server,agent}/root/etc/hologram
mkdir -p /hologram-build/{server,agent}/scripts/
mkdir -p /hologram-build/{server,agent}/root/etc/init.d/

# Copy files needed for the agent package
install -m 0644 ${HOLOGRAM_DIR}/config/agent.json /hologram-build/agent/root/etc/hologram/agent.json
install -m 0755 ${BIN_DIR}/hologram /hologram-build/agent/root/usr/local/bin/
install -m 0755 ${BIN_DIR}/hologram-agent /hologram-build/agent/root/usr/local/bin/
install -m 0755 ${BIN_DIR}/hologram-authorize /hologram-build/agent/root/usr/local/bin/
install -m 0755 ${HOLOGRAM_DIR}/agent/support/linux/after-install_{deb,rpm}.sh /hologram-build/agent/scripts/
install -m 0755 ${HOLOGRAM_DIR}/agent/support/linux/before-remove.sh /hologram-build/agent/scripts/
install -m 0755 ${HOLOGRAM_DIR}/agent/support/linux/init.sh /hologram-build/agent/root/etc/init.d/hologram-agent

# Copy files needed for the server package
install -m 0644 ${HOLOGRAM_DIR}/config/server.json /hologram-build/server/root/etc/hologram/server.json
install -m 0755 ${BIN_DIR}/hologram-server /hologram-build/server/root/usr/local/bin/
install -m 0755 ${HOLOGRAM_DIR}/server/after-install_{deb,rpm}.sh /hologram-build/server/scripts/
install -m 0755 ${HOLOGRAM_DIR}/server/before-remove.sh /hologram-build/server/scripts/
install -m 0755 ${HOLOGRAM_DIR}/server/support/hologram.init.sh /hologram-build/server/root/etc/init.d/hologram-server

ARTIFACTS_DIR=${HOLOGRAM_DIR}/artifacts
mkdir -p ${ARTIFACTS_DIR}

cd /hologram-build/agent
for pkg in deb rpm ; do
    fpm -f -s dir -t ${pkg} -n hologram-agent -v ${GIT_TAG} --after-install /hologram-build/agent/scripts/after-install_${pkg}.sh --before-remove /hologram-build/agent/scripts/before-remove.sh --config-files /etc/hologram/agent.json -C /hologram-build/agent/root -p ${ARTIFACTS_DIR}/hologram-${GIT_TAG}.${pkg} -a amd64 . || exit ${ERRLINPKG}
done

cd /hologram-build/server
for pkg in deb rpm ; do
    fpm -f -s dir -t ${pkg} -n hologram-server -v ${GIT_TAG} --after-install /hologram-build/server/scripts/after-install_${pkg}.sh --before-remove /hologram-build/server/scripts/before-remove.sh --config-files /etc/hologram/server.json -C /hologram-build/server/root -p ${ARTIFACTS_DIR}/hologram-server-${GIT_TAG}.${pkg} -a amd64 . || exit ${ERRLINPKG}
done
