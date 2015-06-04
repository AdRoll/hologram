#!/bin/bash
set -e

if [ "$#" -ne 1 ]; then
    echo "Usage: build-container.sh DOCKER_REGISTRY"
    echo "Example: build-container.sh my.docker.registry.example.com:5000"
    exit 1
fi

CONTAINER_TAG=$(git describe --tags --long)
cp ../../artifacts/hologram-server-${CONTAINER_TAG}.deb objects/hologram-server.deb
REGISTRY=$1
CONTAINER_NAME=${REGISTRY}/hologram_server

docker build -t ${CONTAINER_NAME}:${CONTAINER_TAG} .
docker tag ${CONTAINER_NAME}:${CONTAINER_TAG} ${CONTAINER_NAME}:latest

echo "To push your container: docker push ${CONTAINER_NAME}:${CONTAINER_TAG}"
