#!/bin/bash

# Compile hologram and build packages for all supported platforms

compile_hologram.sh --deps || exit $?
build_linux_pkgs.sh --no-compile || exit $?
build_osx_pkgs.sh --no-compile || exit $?
