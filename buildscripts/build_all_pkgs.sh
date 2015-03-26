#!/bin/bash

compile_hologram.sh --deps || exit 1
build_deb_pkgs.sh --no-compile || exit 1
build_osx_pkgs.sh --no-compile || exit 1
