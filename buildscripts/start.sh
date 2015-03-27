#!/bin/bash

source ${HOLOGRAM_DIR}/buildscripts/returncodes.sh

rsyslogd  # Believe it or not you need syslog to test hologram

if [ "$1" == "build_deb" ]; then
    build_deb_pkgs.sh || exit $?
elif [ "$1" == "build_osx" ]; then
    build_osx_pkgs.sh || exit $?
elif [ "$1" == "build_all" ]; then
    build_all_pkgs.sh || exit $?
elif [ "$1" == "test" ]; then
    compile_hologram.sh --deps || exit $?
elif [ "$1" == "console" ]; then
    install_deps.sh || exit $?
    bash
else
    echo "Valid options: build_deb, build_osx, build_all, test, console"
    exit ${ERRARGS}
fi
