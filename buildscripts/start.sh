#!/bin/bash

source ${HOLOGRAM_DIR}/buildscripts/returncodes.sh

rsyslogd  # Believe it or not you need syslog to test hologram

export HOLOGRAM_HOST="$2"

if [ "$1" == "build_linux" ]; then
    compile_hologram.sh --deps || exit $?
    build_linux_pkgs.sh || exit $?
elif [ "$1" == "build_osx" ]; then
    compile_hologram.sh --deps || exit $?
    build_osx_pkgs.sh || exit $?
elif [ "$1" == "build_all" ]; then
    build_all_pkgs.sh || exit $?
elif [ "$1" == "test" ]; then
    compile_hologram.sh --deps || exit $?
elif [ "$1" == "console" ]; then
    bash
else
    echo "Valid options: build_linux, build_osx, build_all, test, console"
    exit ${ERRARGS}
fi
