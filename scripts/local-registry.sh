#!/usr/bin/env bash
set -xeu

command -v docker >/dev/null 2>&1 || { echo "docker is not installed.  Aborting." >&2; exit 1; }

REGISTRY_NAME=registry
REGISTRY_PORT=5000
IMAGE_NAME=vault-kubernetes-kms

echo "====> create kind docker network"
docker network create kind || true

echo "====> creating registry container unless it already exists"
[[ $(docker ps -f "name=${REGISTRY_NAME}" --format '{{.Names}}') == $REGISTRY_NAME ]] || docker run -d --restart=always -p "${REGISTRY_PORT}:5000" --name "${REGISTRY_NAME}" registry:2

echo "====> building container ..."
docker build --no-cache -t "${IMAGE_NAME}:latest" .

echo "====> tagging container ..."
docker tag "${IMAGE_NAME}:latest" "localhost:${REGISTRY_PORT}/${IMAGE_NAME}:latest"

echo "====> pushing container to local registry ...."
docker push "localhost:${REGISTRY_PORT}/${IMAGE_NAME}:latest"

echo "====> connecting registry to kind ...."
docker network connect kind "${REGISTRY_NAME}" || true
