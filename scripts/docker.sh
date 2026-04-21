#!/bin/bash

set -e

# Note: the set [-/+] x is purely there to turn on and off outputting of the commands being executed.
if [ "${DEBUG}" = "true" ]; then
  set -x
fi

case "$1" in

"buildx")
  docker buildx build \
    --platform linux/amd64,linux/arm64 \
    --target=final \
    --build-arg VENDOR \
    --build-arg BASE_IMAGE_VERSION=latest \
    --build-arg NAME="${BASE_NAME}" \
    --build-arg VERSION="$(scripts/tools.sh version)" \
    --build-arg BUILD_TIME="$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
    -t "${IMAGE_NAME}:latest" \
    -t "${IMAGE_NAME}:$(scripts/tools.sh version)" \
    --push \
    -f Dockerfile .
  ;;

"build")
  docker image build \
    --target=final \
    --build-arg VENDOR \
    --build-arg BASE_IMAGE_VERSION=latest \
    --build-arg NAME="${BASE_NAME}" \
    --build-arg VERSION=latest \
    --build-arg BUILD_TIME="$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
    -t "${IMAGE_NAME}:latest" \
    -f Dockerfile .
  ;;

*)
  echo "error: incorrect '$1' command..."
  ;;

esac
