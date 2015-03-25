#!/bin/bash

GIT_TAG=$(git describe --tags --long | sed 's/-/\./' | sed 's/-g/-/' | sed 's/-/~/')

# Avoid using ssh to get the repos
git config --global url."https://github.com/".insteadOf "git@github.com:"

source gvp
gpm install

rsyslogd  # Believe it or not you need syslog to test hologram
go test -v ./... || exit 1

BIN_DIR=/go/src/github.com/AdRoll/hologram/.godeps/bin

GOOS=linux  go install ./... || exit 1
GOOS=darwin go install ./... || exit 1

mkdir -p /hologram-build/{server,agent}/root/usr/local/bin
mkdir -p /hologram-build/{server,agent}/root/etc/hologram
mkdir -p /hologram-build/{server,agent}/scripts/
mkdir -p /hologram-build/{server,agent}/root/etc/init.d/

cp ./config/agent.json /hologram-build/agent/root/etc/hologram/agent.json
cp ${BIN_DIR}/hologram-cli /hologram-build/agent/root/usr/local/bin/
cp ${BIN_DIR}/hologram-agent /hologram-build/agent/root/usr/local/bin/
cp ${BIN_DIR}/hologram-authorize /hologram-build/agent/root/usr/local/bin/

cp ./agent/support/debian/after-install.sh /hologram-build/agent/scripts/
cp ./agent/support/debian/before-remove.sh /hologram-build/agent/scripts/
cp ./agent/support/debian/init.sh /hologram-build/agent/root/etc/init.d/hologram-agent

cp ./config/server.json /hologram-build/server/root/etc/hologram/server.json

cp ${BIN_DIR}/hologram-cli /hologram-build/server/root/usr/local/bin/
cp ${BIN_DIR}/hologram-server /hologram-build/server/root/usr/local/bin/
cp ${BIN_DIR}/hologram-authorize /hologram-build/server/root/usr/local/bin/

cp ./server/after-install.sh /hologram-build/server/scripts/
cp ./server/before-remove.sh /hologram-build/server/scripts/

cp ./server/support/hologram.init.sh /hologram-build/server/root/etc/init.d/hologram-server

ARTIFACTS_DIR=/go/src/github.com/AdRoll/hologram/artifacts
mkdir -p ${ARTIFACTS_DIR}

cd /hologram-build/agent
fpm -f -s dir -t deb -n hologram-agent -v ${GIT_TAG}  --after-install /hologram-build/agent/scripts/after-install.sh  --before-remove /hologram-build/agent/scripts/before-remove.sh  --config-files /etc/hologram/agent.json  -C /hologram-build/agent/root  -p ${ARTIFACTS_DIR}/hologram-${GIT_TAG}.deb        -a amd64 .  || exit 1

cd /hologram-build/server
fpm -f -s dir -t deb -n hologram-server -v ${GIT_TAG} --after-install /hologram-build/server/scripts/after-install.sh --before-remove /hologram-build/server/scripts/before-remove.sh --config-files /etc/hologram/server.json -C /hologram-build/server/root -p ${ARTIFACTS_DIR}/hologram-server-${GIT_TAG}.deb -a amd64 . || exit 1

mkdir -p /hologram-build/darwin/{root,scripts}
mkdir -p /hologram-build/darwin/root/usr/bin/
mkdir -p /hologram-build/darwin/root/etc/hologram/
mkdir -p /hologram-build/darwin/root/Library/LaunchDaemons
mkdir -p /hologram-build/darwin/scripts
mkdir -p /hologram-build/darwin/flat/base.pkg/

cp ${BIN_DIR}/darwin_amd64/hologram-{agent,cli,authorize,boot} /hologram-build/darwin/root/usr/bin/
cp /go/src/github.com/AdRoll/hologram/config/agent.json /hologram-build/darwin/root/etc/hologram/agent.json
cp /go/src/github.com/AdRoll/hologram/agent/support/darwin/com.adroll.hologram* /hologram-build/darwin/root/Library/LaunchDaemons/
cp /go/src/github.com/AdRoll/hologram/agent/support/darwin/postinstall.sh /hologram-build/darwin/scripts/postinstall
chmod +x /hologram-build/darwin/scripts/postinstall

NUM_FILES=$(find /hologram-build/darwin/root | wc -l)
INSTALL_KB_SIZE=$(du -k -s /hologram-build/darwin/root | awk '{print $1}')

cat <<EOF > /hologram-build/darwin/flat/base.pkg/PackageInfo
<?xml version="1.0" encoding="utf-8" standalone="no"?>
<pkg-info overwrite-permissions="true" relocatable="false" identifier="com.adroll.hologram" postinstall-action="none" version="${GIT_TAG}" format-version="2" generator-version="InstallCmds-502 (14B25)" auth="root">
    <payload numberOfFiles="${NUM_FILES}" installKBytes="${INSTALL_KB_SIZE}"/>
    <bundle-version/>
    <upgrade-bundle/>
    <update-bundle/>
    <atomic-update-bundle/>
    <strict-identifier/>
    <relocate/>
    <scripts>
        <postinstall file="./postinstall"/>
    </scripts>
</pkg-info>
EOF

( cd /hologram-build/darwin/root && find . | cpio -o --format odc --owner 0:80 | gzip -c ) > /hologram-build/darwin/flat/base.pkg/Payload
( cd /hologram-build/darwin/scripts && find . | cpio -o --format odc --owner 0:80 | gzip -c ) > /hologram-build/darwin/flat/base.pkg/Scripts
mkbom -u 0 -g 80 root /hologram-build/darwin/flat/base.pkg/Bom
( cd /hologram-build/darwin/flat/base.pkg && /usr/local/bin/xar --compression none -cf "/go/src/github.com/AdRoll/hologram/artifacts/Hologram-${GIT_TAG}.pkg" * ) || exit 1
