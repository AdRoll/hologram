#!/bin/bash
#docker build -t hologram_build .
docker run --rm -t -i -v $(pwd):/go/src/github.com/AdRoll/hologram hologram_build $1
