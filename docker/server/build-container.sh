#!/bin/bash

if [ "$#" -ne 3 ]; then
    echo "Usage: build-container.sh DEB_PKG CONFIG_FILE CONTAINER_NAME CONTAINER_TAG"
    echo "Example: build-container.sh ../../artifacts/hologram-server-1.1.83\~da8984f.deb my.docker.registry.example.com:5000/hologram_server staging"
    exit 1
fi

rm objects/hologram-server.deb
HOLOGRAM_PKG=$1
CONTAINER_NAME=$2
CONTAINER_TAG=$3

cp ${HOLOGRAM_PKG} objects/hologram-server.deb
docker build -t ${CONTAINER_NAME}:${CONTAINER_TAG} .

echo "To push your container: docker push ${CONTAINER_NAME}:${CONTAINER_TAG}"
