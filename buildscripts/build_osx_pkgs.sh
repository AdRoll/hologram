#!/bin/bash

source ${HOLOGRAM_DIR}/buildscripts/returncodes.sh

cd ${HOLOGRAM_DIR} && export GIT_TAG=$(git describe --tags --long)

if [ "$1" != "--no-compile" ]; then
    compile_hologram.sh || exit $?
fi

mkdir -p /hologram-build/darwin/{root,scripts}
mkdir -p /hologram-build/darwin/root/usr/local/bin/
mkdir -p /hologram-build/darwin/root/etc/hologram/
mkdir -p /hologram-build/darwin/root/Library/LaunchDaemons
mkdir -p /hologram-build/darwin/root/Library/LaunchAgents
mkdir -p /hologram-build/darwin/scripts
mkdir -p /hologram-build/darwin/flat/base.pkg/

install -m 0755 ${BIN_DIR}/darwin_amd64/hologram{-agent,,-authorize,-boot} /hologram-build/darwin/root/usr/local/bin/
install -m 0644 ${HOLOGRAM_DIR}/config/agent.json /hologram-build/darwin/root/etc/hologram/agent.json
install -m 0644 ${HOLOGRAM_DIR}/agent/support/darwin/com.adroll.hologram{-ip,}.plist /hologram-build/darwin/root/Library/LaunchDaemons/
install -m 0644 ${HOLOGRAM_DIR}/agent/support/darwin/com.adroll.hologram-me.plist /hologram-build/darwin/root/Library/LaunchAgents/
install -m 0755 ${HOLOGRAM_DIR}/agent/support/darwin/postinstall.sh /hologram-build/darwin/scripts/postinstall
install -m 0755 ${HOLOGRAM_DIR}/agent/support/darwin/preinstall.sh /hologram-build/darwin/scripts/preinstall

# Special handling for custom host override
if [[ -n "$HOLOGRAM_HOST" ]]
then
  cat "${HOLOGRAM_DIR}/config/agent.json" | jq ". + {\"host\": \"$HOLOGRAM_HOST\"}" > /hologram-build/darwin/root/etc/hologram/agent.json
fi

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
        <preinstall file="./preinstall"/>
        <postinstall file="./postinstall"/>
    </scripts>
</pkg-info>
EOF

PKG_LOCATION="artifacts/Hologram-${GIT_TAG}.pkg"

( cd /hologram-build/darwin/root && find . | cpio -o --format odc --owner 0:80 | gzip -c ) > /hologram-build/darwin/flat/base.pkg/Payload
( cd /hologram-build/darwin/scripts && find . | cpio -o --format odc --owner 0:80 | gzip -c ) > /hologram-build/darwin/flat/base.pkg/Scripts
mkbom -u 0 -g 80 /hologram-build/darwin/root /hologram-build/darwin/flat/base.pkg/Bom || exit ${ERROSXPKG}
( cd /hologram-build/darwin/flat/base.pkg && /usr/local/bin/xar --compression none -cf "${HOLOGRAM_DIR}/${PKG_LOCATION}" * ) || exit ${ERROSXPKG}
echo "osx package has been built: ${PKG_LOCATION}"
