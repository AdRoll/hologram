#!/bin/bash

rsyslogd  # Believe it or not you need syslog to test hologram

if [ "$1" == "build_deb" ]; then
    build_deb_pkgs.sh || exit 1
elif [ "$1" == "build_osx" ]; then
    build_osx_pkgs.sh || exit 1
elif [ "$1" == "build_all" ]; then
    build_all_pkgs.sh || exit 1
elif [ "$1" == "test" ]; then
    compile_hologram.sh
elif [ "$1" == "console" ]; then
    install_deps.sh || exit 1
    bash
else
    echo "Valid options: build_deb, build_osx, build_all, test, console"
    exit 1
fi
