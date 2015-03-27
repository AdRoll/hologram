#!/bin/bash

source ${HOLOGRAM_DIR}/buildscripts/returncodes.sh

cd ${HOLOGRAM_DIR}
gpm install || exit ${ERRDEPINST}
