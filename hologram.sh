#!/bin/bash
docker build -t adroll/hologram_env .
docker run --rm -t -i -v $(pwd):/go/src/github.com/AdRoll/hologram adroll/hologram_env $1 $2
