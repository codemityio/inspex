ARG VENDOR="vendor"
ARG BASE_IMAGE_VERSION="latest"

FROM ${VENDOR}/golang:${BASE_IMAGE_VERSION} AS build

ARG VENDOR="vendor"
ARG NAME="app"
ARG VERSION=""
ARG BUILD_TIME=""

WORKDIR /tmp/build

COPY "cmd" "cmd"
COPY "internal" "internal"
COPY "pkg" "pkg"
COPY "go.*" "."
COPY "*.go" "."

RUN go build \
  -ldflags "\
-X 'main.name=${NAME}' \
-X 'main.version=${VERSION}' \
-X 'main.copyright=${VENDOR}' \
-X 'main.authorName=${VENDOR}' \
-X 'main.buildTime=${BUILD_TIME}'\
" -o bin/app . \
    && go clean -cache -modcache -testcache

FROM ${VENDOR}/alpine:${BASE_IMAGE_VERSION} AS final

WORKDIR /opt/app

ENV PATH="/opt/app/bin:${PATH}"

COPY --from=build /tmp/build/bin/app /opt/app/bin/app

COPY entrypoint.sh /

RUN adduser -D -h /home/commander -s /bin/bash commander \
    && chown -R commander:commander /opt/app

RUN ["chmod", "+x", "/entrypoint.sh"]

USER commander

VOLUME /opt/app/var

ENTRYPOINT ["/entrypoint.sh"]
