#!/bin/bash

# Perform static code analysis and serve godoc locally
cd ${HOLOGRAM_DIR}; godoc -analysis=type  -http=:6060
